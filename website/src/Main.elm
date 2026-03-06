module Main exposing (main)

import Browser
import Element exposing (Attribute, Element, centerX, clipX, column, el, fill, height, html, htmlAttribute, link, maximum, minimum, newTabLink, padding, paddingEach, paragraph, px, rgb255, row, scrollbarY, spacing, text, width, wrappedRow)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Html
import Html.Attributes as HtmlAttr


type alias Model =
    {}


type Msg
    = NoOp


main : Program () Model Msg
main =
    Browser.document
        { init = \_ -> ( {}, Cmd.none )
        , update = update
        , subscriptions = \_ -> Sub.none
        , view = view
        }


update : Msg -> Model -> ( Model, Cmd Msg )
update _ model =
    ( model, Cmd.none )


view : Model -> Browser.Document Msg
view _ =
    { title = "Belm Language"
    , body =
        [ Element.layout
            [ Background.color (rgb255 244 248 255)
            , Font.family
                [ Font.typeface "IBM Plex Sans"
                , Font.typeface "Helvetica Neue"
                , Font.sansSerif
                ]
            , Font.color (rgb255 26 41 59)
            ]
            page
        ]
    }


page : Element Msg
page =
    column
        [ width fill
        , spacing 20
        , paddingEach { top = 20, right = 20, bottom = 28, left = 20 }
        ]
        [ topBar
        , hero
        , codeExample
        , install
        , quickStart
        , features
        , audience
        , docsAndExamples
        ]


topBar : Element Msg
topBar =
    panel
        [ row [ width fill ]
            [ el [ Font.size 28, Font.bold, Font.color (rgb255 22 57 96) ] (text "Belm")
            , el [ width fill ] (text "")
            ]
        ]


hero : Element Msg
hero =
    panel
        [ column [ spacing 10, width fill ]
            [ paragraph [ Font.size 42, Font.bold, Font.color (rgb255 16 44 79), width (fill |> maximum 900) ]
                [ text "A simple declarative backend language." ]
            , paragraph [ Font.size 19, Font.color (rgb255 72 95 123), width (fill |> maximum 880) ]
                [ text "Belm compiles declarative source into a self-contained server executable with API, auth, admin panel, monitoring, and backups."
                ]
            , paragraph [ Font.size 16, Font.color (rgb255 96 116 140), width (fill |> maximum 880) ]
                [ text "Inspired by "
                , newTabLink
                    [ Font.color (rgb255 36 82 132)
                    , Font.semiBold
                    , htmlAttribute (HtmlAttr.style "cursor" "pointer")
                    ]
                    { url = "https://elm-lang.org"
                    , label = text "Elm"
                    }
                , text " and "
                , newTabLink
                    [ Font.color (rgb255 36 82 132)
                    , Font.semiBold
                    , htmlAttribute (HtmlAttr.style "cursor" "pointer")
                    ]
                    { url = "https://pocketbase.io"
                    , label = text "PocketBase"
                    }
                , text "."
                ]
            , row [ spacing 10, paddingEach { top = 6, right = 0, bottom = 0, left = 0 } ]
                [ primaryButton "Get Started" "docs/viewer.html?doc=getting-started"
                , secondaryButton "Open Advanced Guide" "docs/viewer.html?doc=advanced"
                ]
            ]
        ]


codeExample : Element Msg
codeExample =
    panel
        [ sectionTitle "Belm Syntax Example"
        , codeBlock
        ]


install : Element Msg
install =
    panel
        [ sectionTitle "Install"
        , downloadInstallRow
        , pathInstallRow
        , commandRow "3" "Check" "belm version"
        , pluginInstallRow
        ]


downloadInstallRow : Element Msg
downloadInstallRow =
    row
        [ width fill
        , spacing 10
        , Background.color (rgb255 245 250 255)
        , Border.width 1
        , Border.color (rgb255 213 225 241)
        , Border.rounded 10
        , paddingEach { top = 10, right = 12, bottom = 10, left = 12 }
        ]
        [ el
            [ Font.bold
            , Font.size 15
            , Font.color (rgb255 34 76 122)
            , Background.color (rgb255 224 236 252)
            , Border.rounded 999
            , paddingEach { top = 3, right = 8, bottom = 3, left = 8 }
            ]
            (text "1")
        , el [ Font.bold, Font.size 18, width (px 104), Font.color (rgb255 28 66 108) ] (text "Download")
        , newTabLink
            [ Font.size 16
            , Font.semiBold
            , Font.color (rgb255 36 82 132)
            , htmlAttribute (HtmlAttr.style "cursor" "pointer")
            ]
            { url = "https://github.com/marciofrayze/mar/releases"
            , label = text "github.com/marciofrayze/mar/releases"
            }
        ]


pathInstallRow : Element Msg
pathInstallRow =
    column
        [ spacing 10
        , width fill
        , Background.color (rgb255 245 250 255)
        , Border.width 1
        , Border.color (rgb255 213 225 241)
        , Border.rounded 10
        , paddingEach { top = 10, right = 12, bottom = 10, left = 12 }
        ]
        [ row [ width fill, spacing 10 ]
            [ el
                [ Font.bold
                , Font.size 15
                , Font.color (rgb255 34 76 122)
                , Background.color (rgb255 224 236 252)
                , Border.rounded 999
                , paddingEach { top = 3, right = 8, bottom = 3, left = 8 }
                ]
                (text "2")
            , el [ Font.bold, Font.size 18, width (px 104), Font.color (rgb255 28 66 108) ] (text "Path")
            , instructionText "Move belm to a directory in your PATH."
            ]
        , column
            [ width fill
            , spacing 8
            , paddingEach { top = 0, right = 0, bottom = 0, left = 166 }
            ]
            [ row [ spacing 8 ]
                [ el [ Font.size 13, Font.semiBold, Font.color (rgb255 70 93 121), width (px 88) ] (text "macOS/Linux")
                , codeInlineSmall "mv belm /usr/local/bin/belm && chmod +x /usr/local/bin/belm"
                ]
            , row [ spacing 8 ]
                [ el [ Font.size 13, Font.semiBold, Font.color (rgb255 70 93 121), width (px 88) ] (text "Windows")
                , codeInlineSmall "setx PATH \"%PATH%;C:\\Tools\\belm\""
                ]
            ]
        ]


pluginInstallRow : Element Msg
pluginInstallRow =
    column
        [ spacing 10
        , width fill
        , Background.color (rgb255 245 250 255)
        , Border.width 1
        , Border.color (rgb255 213 225 241)
        , Border.rounded 10
        , paddingEach { top = 10, right = 12, bottom = 10, left = 12 }
        ]
        [ row [ width fill, spacing 10 ]
            [ el
                [ Font.bold
                , Font.size 15
                , Font.color (rgb255 34 76 122)
                , Background.color (rgb255 224 236 252)
                , Border.rounded 999
                , paddingEach { top = 3, right = 8, bottom = 3, left = 8 }
                ]
                (text "4")
            , el [ Font.bold, Font.size 18, width (px 120), Font.color (rgb255 28 66 108) ] (text "Code editor")
            , instructionText "Currently, Belm supports only VSCode."
            ]
        , column
            [ spacing 6
            , width fill
            , paddingEach { top = 0, right = 0, bottom = 0, left = 166 }
            ]
            [ paragraph [ Font.size 14, Font.color (rgb255 70 93 121) ]
                [ text "Open VSCode Extensions (Cmd+Shift+X on macOS, Ctrl+Shift+X on Windows/Linux)." ]
            , paragraph [ Font.size 14, Font.color (rgb255 70 93 121) ]
                [ text "Search for "
                , el [ Font.semiBold ] (text "\"Belm Language Support\"")
                , text " and click Install."
                ]
            ]
        , paragraph
            [ Font.size 14
            , Font.color (rgb255 72 95 123)
            , width fill
            , paddingEach { top = 0, right = 0, bottom = 0, left = 166 }
            ]
            [ text "The VSCode extension requires belm on your PATH to start LSP and formatting." ]
        ]


features : Element Msg
features =
    panel
        [ sectionTitle "Why Belm"
        , whyRow
            "Friendly errors"
            "Clear feedback when something is wrong."
            "Actionable compiler and runtime errors."
        , whyRow
            "Secure defaults"
            "Safe behavior without heavy setup."
            "Conservative runtime defaults from day one."
        , whyRow
            "Typed actions"
            "Reliable multi-entity writes."
            "Transactional actions with compile-time checks."
        , whyRow
            "Built-in auth and admin"
            "Core app operations available immediately."
            "Email auth, role checks, admin panel, monitoring, and backups."
        ]


audience : Element Msg
audience =
    panel
        [ sectionTitle "Who Belm Is For"
        , useCaseRow
            "Functional developers"
            "Prefer declarative code and explicit data flow."
            "Belm keeps backend behavior declarative and easy to reason about."
        , useCaseRow
            "One-person projects"
            "Need to build and ship alone with minimal overhead."
            "Belm provides auth, admin, monitoring, and backups in one binary."
        , useCaseRow
            "Small systems"
            "Need a straightforward backend for focused products and internal tools."
            "Belm favors clear, maintainable code over unnecessary complexity."
        ]


quickStart : Element Msg
quickStart =
    panel
        [ sectionTitle "Quick Start"
        , commandRow "1" "Develop" "belm dev examples/store.belm"
        , commandRow "2" "Compile" "belm compile examples/store.belm"
        , commandRow "3" "Deploy" "cd build/store && ./store serve"
        ]


docsAndExamples : Element Msg
docsAndExamples =
    panel
        [ sectionTitle "Resources"
        , wrappedRow [ spacing 12, width fill ]
            [ resourceCard "Getting Started" "docs/viewer.html?doc=getting-started"
            , resourceCard "Advanced Guide" "docs/viewer.html?doc=advanced"
            , resourceCard "Todo Example" "examples/todo.belm"
            , resourceCard "Store Example" "examples/store.belm"
            ]
        ]


panel : List (Element Msg) -> Element Msg
panel children =
    column
        [ width (fill |> maximum 1040)
        , centerX
        , spacing 12
        , padding 16
        , Background.color (rgb255 255 255 255)
        , Border.width 1
        , Border.color (rgb255 209 222 239)
        , Border.rounded 12
        ]
        children


sectionTitle : String -> Element Msg
sectionTitle label =
    paragraph [ Font.size 26, Font.bold, Font.color (rgb255 20 53 89) ] [ text label ]


whyRow : String -> String -> String -> Element Msg
whyRow title text1 text2 =
    row
        [ width fill
        , spacing 14
        , padding 12
        , Background.color (rgb255 246 250 255)
        , Border.width 1
        , Border.color (rgb255 211 224 241)
        , Border.rounded 10
        ]
        [ el
            [ width (px 240)
            , Font.size 18
            , Font.bold
            , Font.color (rgb255 42 58 77)
            ]
            (text title)
        , column [ width fill, spacing 4 ]
            [ paragraph [ Font.size 16, Font.color (rgb255 93 107 126) ] [ text text1 ]
            , paragraph [ Font.size 16, Font.color (rgb255 68 86 108), Font.semiBold ] [ text text2 ]
            ]
        ]


useCaseRow : String -> String -> String -> Element Msg
useCaseRow audienceTitle pain solution =
    row
        [ width fill
        , spacing 14
        , padding 12
        , Background.color (rgb255 246 250 255)
        , Border.width 1
        , Border.color (rgb255 211 224 241)
        , Border.rounded 10
        ]
        [ el
            [ width (px 240)
            , Font.size 18
            , Font.bold
            , Font.color (rgb255 42 58 77)
            ]
            (text audienceTitle)
        , column [ width fill, spacing 4 ]
            [ paragraph [ Font.size 16, Font.color (rgb255 93 107 126) ] [ text pain ]
            , paragraph [ Font.size 16, Font.color (rgb255 68 86 108), Font.semiBold ] [ text solution ]
            ]
        ]


commandRow : String -> String -> String -> Element Msg
commandRow number label command =
    row
        [ width fill
        , spacing 10
        , Background.color (rgb255 245 250 255)
        , Border.width 1
        , Border.color (rgb255 213 225 241)
        , Border.rounded 10
        , paddingEach { top = 10, right = 12, bottom = 10, left = 12 }
        ]
        [ el
            [ Font.bold
            , Font.size 15
            , Font.color (rgb255 34 76 122)
            , Background.color (rgb255 224 236 252)
            , Border.rounded 999
            , paddingEach { top = 3, right = 8, bottom = 3, left = 8 }
            ]
            (text number)
        , el [ Font.bold, Font.size 18, width (px 104), Font.color (rgb255 28 66 108) ] (text label)
        , codeInline command
        ]


resourceCard : String -> String -> Element Msg
resourceCard label target =
    link
        [ width (fill |> minimum 220)
        , Background.color (rgb255 242 248 255)
        , Border.width 1
        , Border.color (rgb255 206 222 242)
        , Border.rounded 10
        , paddingEach { top = 12, right = 12, bottom = 12, left = 12 }
        , Font.size 17
        , Font.color (rgb255 28 71 116)
        , Font.semiBold
        , htmlAttribute (HtmlAttr.style "cursor" "pointer")
        ]
        { url = target
        , label = text label
        }


primaryButton : String -> String -> Element Msg
primaryButton label target =
    link
        (buttonAttributes
            (rgb255 45 126 210)
            (rgb255 245 250 255)
        )
        { url = target
        , label = text label
        }


secondaryButton : String -> String -> Element Msg
secondaryButton label target =
    link
        (buttonAttributes
            (rgb255 230 239 250)
            (rgb255 36 82 132)
        )
        { url = target
        , label = text label
        }


buttonAttributes : Element.Color -> Element.Color -> List (Attribute Msg)
buttonAttributes bg fg =
    [ Background.color bg
    , Font.color fg
    , Border.rounded 10
    , paddingEach { top = 10, right = 14, bottom = 10, left = 14 }
    , Font.semiBold
    , htmlAttribute (HtmlAttr.style "cursor" "pointer")
    ]


navLink : String -> String -> Element Msg
navLink label target =
    link
        [ Font.size 14
        , Font.semiBold
        , Font.color (rgb255 64 88 118)
        , paddingEach { top = 6, right = 0, bottom = 0, left = 14 }
        , htmlAttribute (HtmlAttr.style "cursor" "pointer")
        ]
        { url = target
        , label = text label
        }


codeInline : String -> Element Msg
codeInline source =
    el
        [ Background.color (rgb255 22 43 67)
        , Border.rounded 7
        , paddingEach { top = 7, right = 9, bottom = 7, left = 9 }
        ]
        (el
            [ Font.family [ Font.typeface "IBM Plex Mono", Font.monospace ]
            , Font.size 14
            , Font.color (rgb255 216 231 248)
            ]
            (text source)
        )


codeInlineSmall : String -> Element Msg
codeInlineSmall source =
    el
        [ Background.color (rgb255 22 43 67)
        , Border.rounded 7
        , paddingEach { top = 6, right = 8, bottom = 6, left = 8 }
        ]
        (el
            [ Font.family [ Font.typeface "IBM Plex Mono", Font.monospace ]
            , Font.size 12
            , Font.color (rgb255 216 231 248)
            ]
            (text source)
        )


instructionText : String -> Element Msg
instructionText value =
    paragraph [ Font.size 16, Font.color (rgb255 70 93 121), width fill ] [ text value ]


codeBlock : Element Msg
codeBlock =
    el
        [ width fill
        , height (px 300)
        , clipX
        , scrollbarY
        , Background.color (rgb255 18 38 61)
        , Border.width 1
        , Border.color (rgb255 38 70 105)
        , Border.rounded 10
        , paddingEach { top = 12, right = 14, bottom = 12, left = 14 }
        ]
        (html
            (Html.pre
                [ HtmlAttr.style "margin" "0"
                , HtmlAttr.style "white-space" "pre"
                , HtmlAttr.style "overflow-wrap" "break-word"
                , HtmlAttr.style "font-family" "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, monospace"
                , HtmlAttr.style "font-size" "14px"
                , HtmlAttr.style "line-height" "1.55"
                , HtmlAttr.style "color" "#D8E9FF"
                ]
                codeSnippet
            )
        )


codeSnippet : List (Html.Html msg)
codeSnippet =
    [ codeKeyword "app"
    , Html.text " "
    , codeEntity "TodoApi"
    , Html.text "\n"
    , codeKeyword "port"
    , Html.text " "
    , codeNumber "4100"
    , Html.text "\n"
    , codeKeyword "database"
    , Html.text " "
    , codeString "\"todo.db\""
    , Html.text "\n\n"
    , codeKeyword "entity"
    , Html.text " "
    , codeEntity "Todo"
    , Html.text " "
    , codePunctuation "{"
    , Html.text "\n"
    , Html.text "  "
    , codeField "id"
    , codePunctuation ":"
    , Html.text " "
    , codeType "Int"
    , Html.text " "
    , codeModifier "primary"
    , Html.text " "
    , codeModifier "auto"
    , Html.text "\n"
    , Html.text "  "
    , codeField "title"
    , codePunctuation ":"
    , Html.text " "
    , codeType "String"
    , Html.text "\n"
    , Html.text "  "
    , codeField "done"
    , codePunctuation ":"
    , Html.text " "
    , codeType "Bool"
    , Html.text "\n\n"
    , Html.text "  "
    , codeKeyword "rule"
    , Html.text " "
    , codeString "\"Title must have at least 3 chars\""
    , Html.text " "
    , codeKeyword "when"
    , Html.text " "
    , codeFunction "len"
    , codePunctuation "("
    , codeField "title"
    , codePunctuation ")"
    , Html.text " "
    , codeOperator ">="
    , Html.text " "
    , codeNumber "3"
    , Html.text "\n"
    , Html.text "  "
    , codeKeyword "authorize"
    , Html.text " "
    , codeCrud "list"
    , Html.text " "
    , codeKeyword "when"
    , Html.text " "
    , codeContext "auth_authenticated"
    , Html.text "\n"
    , Html.text "  "
    , codeKeyword "authorize"
    , Html.text " "
    , codeCrud "create"
    , Html.text " "
    , codeKeyword "when"
    , Html.text " "
    , codeContext "auth_authenticated"
    , Html.text "\n"
    , codePunctuation "}"
    , Html.text "\n"
    ]


token : String -> String -> Html.Html msg
token color value =
    Html.span [ HtmlAttr.style "color" color ] [ Html.text value ]


codeKeyword : String -> Html.Html msg
codeKeyword value =
    token "#7AB8FF" value


codeType : String -> Html.Html msg
codeType value =
    token "#4FD1C5" value


codeModifier : String -> Html.Html msg
codeModifier value =
    token "#B7C5D9" value


codeField : String -> Html.Html msg
codeField value =
    token "#DCE8F8" value


codeEntity : String -> Html.Html msg
codeEntity value =
    token "#92C4FF" value


codeCrud : String -> Html.Html msg
codeCrud value =
    token "#93D7FF" value


codeString : String -> Html.Html msg
codeString value =
    token "#F7C97F" value


codeNumber : String -> Html.Html msg
codeNumber value =
    token "#F5A97F" value


codeFunction : String -> Html.Html msg
codeFunction value =
    token "#82E0AA" value


codeContext : String -> Html.Html msg
codeContext value =
    token "#C3D7FF" value


codeOperator : String -> Html.Html msg
codeOperator value =
    token "#D8E9FF" value


codePunctuation : String -> Html.Html msg
codePunctuation value =
    token "#AFC7E6" value
