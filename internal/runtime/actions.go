package runtime

import (
	"fmt"
	"net/http"

	"mar/internal/model"
	"mar/internal/sqlitecli"
	"mar/internal/suggest"
)

// handleAction executes a typed action (with create/update/delete steps) in a single SQL transaction.
func (r *Runtime) handleAction(w http.ResponseWriter, requestID, actionName string, auth authSession, payload map[string]any) error {
	action := r.actionsByName[actionName]
	if action == nil {
		return newAPIError(http.StatusNotFound, "action_not_found", "Action not found")
	}
	alias := r.aliasesByName[action.InputAlias]
	if alias == nil {
		return newAPIError(http.StatusInternalServerError, "action_misconfigured", fmt.Sprintf("Action %s is misconfigured: missing input alias %s", action.Name, action.InputAlias))
	}

	input, err := normalizeActionInput(alias, payload)
	if err != nil {
		return newAPIError(http.StatusBadRequest, "invalid_action_input", err.Error())
	}

	statements := make([]sqlitecli.Statement, 0, len(action.Steps))
	allCreates := true
	for _, step := range action.Steps {
		entity := r.entitiesByName[step.Entity]
		if entity == nil {
			return newAPIError(http.StatusInternalServerError, "action_misconfigured", fmt.Sprintf("Action %s references unknown entity %s", action.Name, step.Entity))
		}

		stepPayload, err := resolveActionStepValues(step, input)
		if err != nil {
			return newAPIError(http.StatusBadRequest, "invalid_action_input", err.Error())
		}

		if step.Kind != "create" {
			allCreates = false
		}
		stmt, err := r.buildActionStatement(requestID, action, entity, step, auth, stepPayload)
		if err != nil {
			return err
		}
		statements = append(statements, stmt)
	}

	if err := r.DB.ExecTxTagged(requestID, statements); err != nil {
		return err
	}

	status := http.StatusOK
	if allCreates {
		status = http.StatusCreated
	}
	r.writeJSON(w, status, map[string]any{
		"ok":     true,
		"action": action.Name,
		"steps":  len(statements),
	})
	return nil
}

func (r *Runtime) buildActionStatement(requestID string, action *model.Action, entity *model.Entity, step model.ActionStep, auth authSession, stepPayload map[string]any) (sqlitecli.Statement, error) {
	switch step.Kind {
	case "create":
		insert, err := buildInsert(entity, stepPayload)
		if err != nil {
			return sqlitecli.Statement{}, newAPIError(http.StatusBadRequest, "invalid_action_input", fmt.Sprintf("Action %s step create %s: %s", action.Name, entity.Name, err.Error()))
		}
		if err := r.ensureAuthorized(entity, "create", auth, insert.Context); err != nil {
			return sqlitecli.Statement{}, err
		}
		if err := r.validateEntityRules(entity, insert.Context); err != nil {
			return sqlitecli.Statement{}, err
		}
		stmt, err := buildInsertStatement(entity, insert)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		return stmt, nil
	case "update":
		id, ok := stepPayload[entity.PrimaryKey]
		if !ok {
			return sqlitecli.Statement{}, newAPIError(http.StatusInternalServerError, "action_misconfigured", fmt.Sprintf("Action %s step update %s is missing primary key %s", action.Name, entity.Name, entity.PrimaryKey))
		}
		row, found, err := r.fetchByID(requestID, entity, id)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		if !found {
			return sqlitecli.Statement{}, newAPIError(http.StatusNotFound, "entity_not_found", entity.Name+" not found")
		}
		current := decodeEntityRow(entity, row)
		updatePayload := map[string]any{}
		for key, value := range stepPayload {
			if key == entity.PrimaryKey {
				continue
			}
			updatePayload[key] = value
		}
		update, err := buildUpdate(entity, updatePayload, current)
		if err != nil {
			return sqlitecli.Statement{}, newAPIError(http.StatusBadRequest, "invalid_action_input", fmt.Sprintf("Action %s step update %s: %s", action.Name, entity.Name, err.Error()))
		}
		if err := r.ensureAuthorized(entity, "update", auth, update.Context); err != nil {
			return sqlitecli.Statement{}, err
		}
		if err := r.validateEntityRules(entity, update.Context); err != nil {
			return sqlitecli.Statement{}, err
		}
		stmt, err := buildUpdateStatement(entity, id, update)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		return stmt, nil
	case "delete":
		id, ok := stepPayload[entity.PrimaryKey]
		if !ok {
			return sqlitecli.Statement{}, newAPIError(http.StatusInternalServerError, "action_misconfigured", fmt.Sprintf("Action %s step delete %s is missing primary key %s", action.Name, entity.Name, entity.PrimaryKey))
		}
		row, found, err := r.fetchByID(requestID, entity, id)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		if !found {
			return sqlitecli.Statement{}, newAPIError(http.StatusNotFound, "entity_not_found", entity.Name+" not found")
		}
		if err := r.ensureAuthorized(entity, "delete", auth, decodeEntityRow(entity, row)); err != nil {
			return sqlitecli.Statement{}, err
		}
		stmt, err := buildDeleteStatement(entity, id)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		return stmt, nil
	default:
		return sqlitecli.Statement{}, newAPIError(http.StatusInternalServerError, "action_misconfigured", fmt.Sprintf("Action %s uses unsupported step kind %s", action.Name, step.Kind))
	}
}

func normalizeActionInput(alias *model.TypeAlias, payload map[string]any) (map[string]any, error) {
	fieldNames := aliasFieldNames(alias)
	for key := range payload {
		if !aliasHasField(alias, key) {
			return nil, fmt.Errorf("unknown input field %q for %s%s", key, alias.Name, suggest.DidYouMeanSuffix(key, fieldNames))
		}
	}
	out := map[string]any{}
	for _, field := range alias.Fields {
		raw, ok := payload[field.Name]
		if !ok {
			return nil, fmt.Errorf("missing required input field %q for %s", field.Name, alias.Name)
		}
		value, err := normalizeActionInputValue(field.Name, field.Type, raw)
		if err != nil {
			return nil, err
		}
		out[field.Name] = value
	}
	return out, nil
}

func aliasHasField(alias *model.TypeAlias, fieldName string) bool {
	for _, f := range alias.Fields {
		if f.Name == fieldName {
			return true
		}
	}
	return false
}

func aliasFieldNames(alias *model.TypeAlias) []string {
	out := make([]string, 0, len(alias.Fields))
	for _, field := range alias.Fields {
		out = append(out, field.Name)
	}
	return out
}

func normalizeActionInputValue(name, typ string, raw any) (any, error) {
	switch typ {
	case "Int":
		n, ok := toInt64(raw)
		if !ok {
			return nil, fmt.Errorf("input.%s must be Int", name)
		}
		return n, nil
	case "Posix":
		n, ok := toInt64(raw)
		if !ok {
			return nil, fmt.Errorf("input.%s must be Posix (Unix milliseconds)", name)
		}
		return n, nil
	case "Float":
		f, ok := toFloat64(raw)
		if !ok {
			return nil, fmt.Errorf("input.%s must be Float", name)
		}
		return f, nil
	case "String":
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("input.%s must be String", name)
		}
		return s, nil
	case "Bool":
		b, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("input.%s must be Bool", name)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported input type %s for input.%s", typ, name)
	}
}

func resolveActionStepValues(step model.ActionStep, input map[string]any) (map[string]any, error) {
	payload := map[string]any{}
	for _, expr := range step.Values {
		value, err := resolveActionExprValue(expr, input)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", expr.Field, err)
		}
		payload[expr.Field] = value
	}
	return payload, nil
}

func resolveActionExprValue(expr model.ActionFieldExpr, input map[string]any) (any, error) {
	switch expr.SourceKind {
	case "input":
		value, ok := input[expr.InputField]
		if !ok {
			return nil, fmt.Errorf("input field %q was not provided", expr.InputField)
		}
		return value, nil
	case "literal_string":
		s, ok := expr.Literal.(string)
		if !ok {
			return nil, fmt.Errorf("invalid string literal")
		}
		return s, nil
	case "literal_bool":
		b, ok := expr.Literal.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid bool literal")
		}
		return b, nil
	case "literal_int":
		n, ok := toInt64(expr.Literal)
		if !ok {
			return nil, fmt.Errorf("invalid int literal")
		}
		return n, nil
	case "literal_float":
		f, ok := toFloat64(expr.Literal)
		if !ok {
			return nil, fmt.Errorf("invalid float literal")
		}
		return f, nil
	case "literal_null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported source %q", expr.SourceKind)
	}
}

func buildInsertStatement(entity *model.Entity, insert *insertBuild) (sqlitecli.Statement, error) {
	table, err := quoteIdentifier(entity.Table)
	if err != nil {
		return sqlitecli.Statement{}, err
	}
	if len(insert.Columns) == 0 {
		return sqlitecli.Statement{
			Query: fmt.Sprintf("INSERT INTO %s DEFAULT VALUES", table),
		}, nil
	}
	cols := make([]string, 0, len(insert.Columns))
	placeholders := make([]string, 0, len(insert.Columns))
	for _, c := range insert.Columns {
		q, err := quoteIdentifier(c)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		cols = append(cols, q)
		placeholders = append(placeholders, "?")
	}
	return sqlitecli.Statement{
		Query: fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, stringsJoin(cols, ", "), stringsJoin(placeholders, ", ")),
		Args:  insert.Values,
	}, nil
}

func buildUpdateStatement(entity *model.Entity, id any, update *updateBuild) (sqlitecli.Statement, error) {
	table, err := quoteIdentifier(entity.Table)
	if err != nil {
		return sqlitecli.Statement{}, err
	}
	pk, err := quoteIdentifier(entity.PrimaryKey)
	if err != nil {
		return sqlitecli.Statement{}, err
	}
	assignments := make([]string, 0, len(update.Columns))
	for _, c := range update.Columns {
		q, err := quoteIdentifier(c)
		if err != nil {
			return sqlitecli.Statement{}, err
		}
		assignments = append(assignments, fmt.Sprintf("%s = ?", q))
	}
	args := append(update.Values, id)
	return sqlitecli.Statement{
		Query: fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, stringsJoin(assignments, ", "), pk),
		Args:  args,
	}, nil
}

func buildDeleteStatement(entity *model.Entity, id any) (sqlitecli.Statement, error) {
	table, err := quoteIdentifier(entity.Table)
	if err != nil {
		return sqlitecli.Statement{}, err
	}
	pk, err := quoteIdentifier(entity.PrimaryKey)
	if err != nil {
		return sqlitecli.Statement{}, err
	}
	return sqlitecli.Statement{
		Query: fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, pk),
		Args:  []any{id},
	}, nil
}

func stringsJoin(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += sep + items[i]
	}
	return out
}
