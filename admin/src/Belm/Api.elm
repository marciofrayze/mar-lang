module Belm.Api exposing
    ( AuthInfo
    , ActionInfo
    , InputAliasInfo
    , InputAliasField
    , SystemAuthInfo
    , Entity
    , Field
    , FieldType(..)
    , Row
    , Schema
    , decodeRows
    , decodeSchema
    , encodePayload
    , fieldTypeLabel
    , rowDecoder
    , valueToString
    )

import Dict exposing (Dict)
import Json.Decode as Decode exposing (Decoder, Value)
import Json.Encode as Encode
import String


type FieldType
    = IntType
    | StringType
    | BoolType
    | FloatType


type alias Field =
    { name : String
    , fieldType : FieldType
    , primary : Bool
    , auto : Bool
    , optional : Bool
    }


type alias Entity =
    { name : String
    , table : String
    , resource : String
    , primaryKey : String
    , fields : List Field
    }


type alias Schema =
    { appName : String
    , portNumber : Int
    , database : String
    , entities : List Entity
    , auth : Maybe AuthInfo
    , systemAuth : Maybe SystemAuthInfo
    , inputAliases : List InputAliasInfo
    , actions : List ActionInfo
    }


type alias Row =
    Dict String Value


type alias AuthInfo =
    { enabled : Bool
    , userEntity : String
    , emailField : String
    , roleField : String
    , emailTransport : String
    , needsBootstrap : Bool
    }


type alias SystemAuthInfo =
    { enabled : Bool
    , emailTransport : String
    , needsBootstrap : Bool
    }


type alias ActionInfo =
    { name : String
    , inputAlias : String
    , steps : Int
    }


type alias InputAliasInfo =
    { name : String
    , fields : List InputAliasField
    }


type alias InputAliasField =
    { name : String
    , fieldType : String
    }


decodeSchema : Decoder Schema
decodeSchema =
    Decode.map8 Schema
        (Decode.field "appName" Decode.string)
        (Decode.field "port" Decode.int)
        (Decode.field "database" Decode.string)
        (Decode.field "entities" (Decode.list entityDecoder))
        decodeAuthInfo
        decodeSystemAuthInfo
        decodeInputAliases
        decodeActionInfo


decodeAuthInfo : Decoder (Maybe AuthInfo)
decodeAuthInfo =
    Decode.oneOf
        [ Decode.field "auth" (Decode.map Just authInfoDecoder)
        , Decode.succeed Nothing
        ]


authInfoDecoder : Decoder AuthInfo
authInfoDecoder =
    Decode.map6 AuthInfo
        (Decode.field "enabled" Decode.bool)
        (Decode.field "userEntity" Decode.string)
        (Decode.field "emailField" Decode.string)
        (Decode.field "roleField" Decode.string)
        (Decode.field "emailTransport" Decode.string)
        (Decode.oneOf
            [ Decode.field "needsBootstrap" Decode.bool
            , Decode.succeed False
            ]
        )


decodeSystemAuthInfo : Decoder (Maybe SystemAuthInfo)
decodeSystemAuthInfo =
    Decode.oneOf
        [ Decode.field "systemAuth" (Decode.map Just systemAuthInfoDecoder)
        , Decode.succeed Nothing
        ]


systemAuthInfoDecoder : Decoder SystemAuthInfo
systemAuthInfoDecoder =
    Decode.map3 SystemAuthInfo
        (Decode.field "enabled" Decode.bool)
        (Decode.field "emailTransport" Decode.string)
        (Decode.oneOf
            [ Decode.field "needsBootstrap" Decode.bool
            , Decode.succeed False
            ]
        )


decodeActionInfo : Decoder (List ActionInfo)
decodeActionInfo =
    Decode.oneOf
        [ Decode.field "actions" (Decode.list actionInfoDecoder)
        , Decode.succeed []
        ]


decodeInputAliases : Decoder (List InputAliasInfo)
decodeInputAliases =
    Decode.oneOf
        [ Decode.field "inputAliases" (Decode.list inputAliasDecoder)
        , Decode.succeed []
        ]


inputAliasDecoder : Decoder InputAliasInfo
inputAliasDecoder =
    Decode.map2 InputAliasInfo
        (Decode.field "name" Decode.string)
        (Decode.field "fields" (Decode.list inputAliasFieldDecoder))


inputAliasFieldDecoder : Decoder InputAliasField
inputAliasFieldDecoder =
    Decode.map2 InputAliasField
        (Decode.field "name" Decode.string)
        (Decode.field "type" Decode.string)


actionInfoDecoder : Decoder ActionInfo
actionInfoDecoder =
    Decode.map3 ActionInfo
        (Decode.field "name" Decode.string)
        (Decode.field "inputAlias" Decode.string)
        (Decode.field "steps" Decode.int)


entityDecoder : Decoder Entity
entityDecoder =
    Decode.map5 Entity
        (Decode.field "name" Decode.string)
        (Decode.field "table" Decode.string)
        (Decode.field "resource" Decode.string)
        (Decode.field "primaryKey" Decode.string)
        (Decode.field "fields" (Decode.list fieldDecoder))


fieldDecoder : Decoder Field
fieldDecoder =
    Decode.map5 Field
        (Decode.field "name" Decode.string)
        (Decode.field "type" Decode.string |> Decode.andThen decodeFieldType)
        (Decode.field "primary" Decode.bool)
        (Decode.field "auto" Decode.bool)
        (Decode.field "optional" Decode.bool)


decodeFieldType : String -> Decoder FieldType
decodeFieldType raw =
    case raw of
        "Int" ->
            Decode.succeed IntType

        "String" ->
            Decode.succeed StringType

        "Bool" ->
            Decode.succeed BoolType

        "Float" ->
            Decode.succeed FloatType

        _ ->
            Decode.fail ("Unknown field type: " ++ raw)


rowDecoder : Decoder Row
rowDecoder =
    Decode.dict Decode.value


decodeRows : Decoder (List Row)
decodeRows =
    Decode.list rowDecoder


fieldTypeLabel : FieldType -> String
fieldTypeLabel fieldType =
    case fieldType of
        IntType ->
            "Int"

        StringType ->
            "String"

        BoolType ->
            "Bool"

        FloatType ->
            "Float"


encodePayload : { forUpdate : Bool } -> List Field -> Dict String String -> Result String Encode.Value
encodePayload options fields valuesByName =
    fields
        |> List.filter (\field -> not field.primary)
        |> List.foldl (encodeField options valuesByName) (Ok [])
        |> Result.map Encode.object


encodeField : { forUpdate : Bool } -> Dict String String -> Field -> Result String (List ( String, Encode.Value )) -> Result String (List ( String, Encode.Value ))
encodeField options valuesByName field partialResult =
    case partialResult of
        Err message ->
            Err message

        Ok items ->
            let
                rawValue =
                    Dict.get field.name valuesByName
                        |> Maybe.withDefault ""
                        |> String.trim
            in
            if rawValue == "" then
                if field.optional || options.forUpdate then
                    Ok items

                else
                    Err ("Field " ++ field.name ++ " is required")

            else
                case encodeByType field rawValue of
                    Ok encoded ->
                        Ok (( field.name, encoded ) :: items)

                    Err message ->
                        Err message


encodeByType : Field -> String -> Result String Encode.Value
encodeByType field rawValue =
    case field.fieldType of
        StringType ->
            Ok (Encode.string rawValue)

        IntType ->
            case String.toInt rawValue of
                Just value ->
                    Ok (Encode.int value)

                Nothing ->
                    Err ("Field " ++ field.name ++ " expects Int")

        FloatType ->
            case String.toFloat rawValue of
                Just value ->
                    Ok (Encode.float value)

                Nothing ->
                    Err ("Field " ++ field.name ++ " expects Float")

        BoolType ->
            let
                lowered =
                    String.toLower rawValue
            in
            if lowered == "true" || lowered == "1" || lowered == "yes" then
                Ok (Encode.bool True)

            else if lowered == "false" || lowered == "0" || lowered == "no" then
                Ok (Encode.bool False)

            else
                Err ("Field " ++ field.name ++ " expects Bool (true/false)")


valueToString : Value -> String
valueToString value =
    case Decode.decodeValue Decode.string value of
        Ok text ->
            text

        Err _ ->
            case Decode.decodeValue Decode.bool value of
                Ok boolValue ->
                    if boolValue then
                        "true"

                    else
                        "false"

                Err _ ->
                    case Decode.decodeValue Decode.float value of
                        Ok number ->
                            if isWhole number then
                                String.fromInt (round number)

                            else
                                String.fromFloat number

                        Err _ ->
                            case Decode.decodeValue (Decode.null "null") value of
                                Ok nullText ->
                                    nullText

                                Err _ ->
                                    Encode.encode 0 value


isWhole : Float -> Bool
isWhole number =
    toFloat (round number) == number
