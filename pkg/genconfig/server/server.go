package server

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type GenConfigServer struct {
	YamlInfo *YamlInfo
}

func (g *GenConfigServer) GenYaml(command *cobra.Command) error {
	err := g.genYaml(command)
	if err != nil {
		return err
	}
	for _, c := range command.Commands() {
		err := g.genYaml(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GenConfigServer) genYaml(command *cobra.Command) error {
	doc := cmdDoc{}

	doc.Name = command.Name()
	doc.Synopsis = forceMultiLine(command.Short)
	doc.Description = forceMultiLine(command.Long)

	flags := command.NonInheritedFlags()
	if flags.HasFlags() {
		doc.Options = genFlagResult(flags)
	}
	flags = command.InheritedFlags()
	if flags.HasFlags() {
		doc.InheritedOptions = genFlagResult(flags)
	}

	if len(command.Example) > 0 {
		doc.Example = command.Example
	}

	if len(command.Commands()) > 0 || len(g.YamlInfo.ParentName) > 0 {
		result := []string{}
		if len(g.YamlInfo.ParentName) > 0 {
			result = append(result, g.YamlInfo.ParentName)
		}
		for _, c := range command.Commands() {
			result = append(result, c.Name())
		}
		doc.SeeAlso = result
	}

	final, err := yaml.Marshal(&doc)
	if err != nil {
		return err
	}

	var filename string

	if g.YamlInfo.ParentName == "" {
		filename = filepath.Join(g.YamlInfo.DocsDir, doc.Name+".yaml")
	} else {
		filename = filepath.Join(g.YamlInfo.DocsDir, g.YamlInfo.ParentName+"_"+doc.Name+".yaml")
	}

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(final)
	if err != nil {
		return err
	}

	return nil
}

func genFlagResult(flags *pflag.FlagSet) []cmdOption {
	result := []cmdOption{}

	flags.VisitAll(func(flag *pflag.Flag) {
		// Todo, when we mark a shorthand is deprecated, but specify an empty message.
		// The flag.ShorthandDeprecated is empty as the shorthand is deprecated.
		// Using len(flag.ShorthandDeprecated) > 0 can't handle this, others are ok.
		if !(len(flag.ShorthandDeprecated) > 0) && len(flag.Shorthand) > 0 {
			opt := cmdOption{
				flag.Name,
				flag.Shorthand,
				flag.DefValue,
				forceMultiLine(flag.Usage),
			}
			result = append(result, opt)
		} else {
			opt := cmdOption{
				Name:         flag.Name,
				DefaultValue: forceMultiLine(flag.DefValue),
				Usage:        forceMultiLine(flag.Usage),
			}
			result = append(result, opt)
		}
	})

	return result
}

func forceMultiLine(s string) string {
	if len(s) > 60 && !strings.Contains(s, "\n") {
		s += "\n"
	}

	return s
}
