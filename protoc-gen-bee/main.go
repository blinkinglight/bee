package main

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/blinkinglight/bee/gen/esoptions"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var eventTemplate = template.Must(template.New("event").Parse(`package {{.Package}}


{{range .Events}}
type {{.Name}} struct {

    {{range .Fields}}{{.GoName}}  {{.GoType}} // {{.IsAggregateID}}
    {{end}}
}

{{end}}
`))

var commandTemplate = template.Must(template.New("command").Parse(`package {{.Package}}

{{range .Commands}}
type {{.Name}} struct {
    {{range .Fields}}{{.GoName}} {{.GoType}}
    {{end}}
}

{{end}}
`))

type Field struct {
	GoName        string
	GoType        string
	IsAggregateID bool
}

type Message struct {
	Name      string
	Fields    []Field
	Aggregate string
	EventType string
}

func isAggregateID(field *protogen.Field) bool {
	opts, ok := field.Desc.Options().(*descriptorpb.FieldOptions)
	if !ok || opts == nil {
		return false
	}
	val := proto.GetExtension(opts, esoptions.E_AggregateId)
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}
func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}

			var events []Message
			var commands []Message
			pkg := string(file.GoPackageName)

			// Paprastumo dėlei: Aggregate bus pirmas message (Project).
			var aggregateName string
			if len(file.Messages) > 0 {
				aggregateName = string(file.Messages[0].GoIdent.GoName)
			}

			for _, msg := range file.Messages {
				fields := []Field{}
				for _, f := range msg.Fields {
					fields = append(fields, Field{
						GoName:        string(f.GoName),
						GoType:        goType(f),
						IsAggregateID: isAggregateID(f),
					})
				}
				m := Message{
					Name:      string(msg.GoIdent.GoName),
					Fields:    fields,
					Aggregate: aggregateName,
					EventType: string(msg.GoIdent.GoName) + "ed", // quick hack
				}

				n := m.Name
				// "Command" detection
				if strings.HasPrefix(n, "Create") || strings.HasSuffix(n, "Command") {
					// Command type, event type hardcoded (could be improved)
					if strings.HasPrefix(n, "Create") {
						m.EventType = strings.Replace(n, "Create", "", 1) + "Created"
					}
					commands = append(commands, m)
				} else if strings.HasSuffix(n, "Created") || strings.HasSuffix(n, "Event") {
					events = append(events, m)
				}
			}

			// Write events.go
			var eventsBuf bytes.Buffer
			eventTemplate.Execute(&eventsBuf, map[string]any{
				"Package": pkg,
				"Events":  events,
			})
			eventFile := file.GeneratedFilenamePrefix + "_events.go"
			g := plugin.NewGeneratedFile(eventFile, file.GoImportPath)
			g.P(eventsBuf.String())

			// Write commands.go
			var commandsBuf bytes.Buffer
			commandTemplate.Execute(&commandsBuf, map[string]any{
				"Package":  pkg,
				"Commands": commands,
			})
			cmdFile := file.GeneratedFilenamePrefix + "_commands.go"
			g2 := plugin.NewGeneratedFile(cmdFile, file.GoImportPath)
			g2.P(commandsBuf.String())
		}
		return nil
	})
}

func goType(f *protogen.Field) string {
	switch f.Desc.Kind() {
	case 1: // double
		return "float64"
	case 2: // float
		return "float32"
	case 3, 4, 5, 17, 18: // various ints
		return "int"
	case 9:
		return "string"
	case 8:
		return "bool"
	default:
		return "string" // default fallback
	}
}
