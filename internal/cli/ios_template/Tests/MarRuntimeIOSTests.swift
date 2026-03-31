import XCTest
@testable import MarRuntimeIOS

final class MarRuntimeIOSTests: XCTestCase {
    func testSchemaDecodingAllowsMissingOptionalCollections() throws {
        let json = """
        {
          "appName": "PersonalTodo",
          "port": 4200,
          "database": "/data/personal-todo.db",
          "auth": {
            "emailField": "email",
            "enabled": true,
            "needsBootstrap": false,
            "roleField": "role",
            "userEntity": "User"
          },
          "entities": [
            {
              "name": "Todo",
              "table": "todos",
              "resource": "/todos",
              "primaryKey": "id",
              "fields": [
                {
                  "name": "id",
                  "type": "Int",
                  "primary": true,
                  "auto": true,
                  "optional": false
                }
              ]
            }
          ]
        }
        """

        let schema = try JSONDecoder().decode(Schema.self, from: Data(json.utf8))

        XCTAssertEqual(schema.appName, "PersonalTodo")
        XCTAssertTrue(schema.inputAliases.isEmpty)
        XCTAssertTrue(schema.actions.isEmpty)
        XCTAssertNil(schema.systemAuth)
    }

    func testFieldDecodingDefaultsCurrentUserToFalse() throws {
        let json = """
        {
          "name": "email",
          "type": "String",
          "primary": false,
          "auto": false,
          "optional": false
        }
        """

        let field = try JSONDecoder().decode(Field.self, from: Data(json.utf8))

        XCTAssertEqual(field.name, "email")
        XCTAssertFalse(field.currentUser)
    }

    func testDateFormattingAndParsingRoundTrip() {
        let millis = 1_742_203_200_000.0
        let formatted = MarDateCodec.formatDateInput(milliseconds: millis)
        XCTAssertEqual(formatted, "2025-03-17")
        XCTAssertEqual(MarDateCodec.parseDateInput(formatted), millis)
    }

    func testDateTimeFormattingAndParsingRoundTrip() {
        let millis = 1_742_203_200_000.0
        let formatted = MarDateCodec.formatDateTimeInput(milliseconds: millis)
        XCTAssertEqual(formatted, "2025-03-17T10:00")
        XCTAssertEqual(MarDateCodec.parseDateTimeInput(formatted), millis)
    }

    func testPayloadEncoderSendsNullForOptionalEmptyField() throws {
        let fields = [
            Field(name: "title", fieldType: .string, relationEntity: nil, currentUser: false, primary: false, auto: false, optional: false, defaultValue: nil),
            Field(name: "notes", fieldType: .string, relationEntity: nil, currentUser: false, primary: false, auto: false, optional: true, defaultValue: nil)
        ]

        let payload = try PayloadEncoder.buildPayload(fields: fields, valuesByName: ["title": "Hello", "notes": ""], forUpdate: false)
        XCTAssertEqual(payload["title"], .string("Hello"))
        XCTAssertEqual(payload["notes"], .null)
    }

    func testPayloadEncoderRejectsMissingRequiredField() {
        let fields = [
            Field(name: "name", fieldType: .string, relationEntity: nil, currentUser: false, primary: false, auto: false, optional: false, defaultValue: nil)
        ]

        XCTAssertThrowsError(try PayloadEncoder.buildPayload(fields: fields, valuesByName: [:], forUpdate: false)) { error in
            XCTAssertEqual(error as? PayloadEncodingError, .requiredField("name"))
        }
    }

    func testRowPresentationUsesFriendlyField() {
        let entity = Entity(
            name: "Student",
            table: "students",
            resource: "/students",
            primaryKey: "id",
            fields: [
                Field(name: "id", fieldType: .int, relationEntity: nil, currentUser: false, primary: true, auto: true, optional: false, defaultValue: nil),
                Field(name: "name", fieldType: .string, relationEntity: nil, currentUser: false, primary: false, auto: false, optional: false, defaultValue: nil)
            ]
        )

        let row: Row = [
            "id": .number(1),
            "name": .string("Leandro")
        ]

        XCTAssertEqual(RowPresentation.relatedRowLabel(entity: entity, row: row), "Leandro")
    }

    func testTransportErrorUsesFriendlyTLSMessage() {
        let error = marTransportError(path: "/auth/request-code", error: URLError(.secureConnectionFailed))

        XCTAssertEqual(error.errorDescription, "A secure connection to the server could not be established.")
        if case let .transport(path, _, details) = error {
            XCTAssertEqual(path, "/auth/request-code")
            XCTAssertTrue(details.contains("secureConnectionFailed"))
        } else {
            XCTFail("Expected transport error")
        }
    }

    func testTransportErrorUsesFriendlyDNSMessage() {
        let error = marTransportError(path: "/_mar/schema", error: URLError(.cannotFindHost))

        XCTAssertEqual(error.errorDescription, "The server address could not be resolved.")
        if case let .transport(path, _, details) = error {
            XCTAssertEqual(path, "/_mar/schema")
            XCTAssertTrue(details.contains("cannotFindHost"))
        } else {
            XCTFail("Expected transport error")
        }
    }
}
