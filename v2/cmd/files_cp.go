package cmd

import (
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "Copy files from A to B within your remote server",
	Long: `
DESCRIPTION

  Copy files from one location to another *within* a *single* Pydio Cells instance. 
  To copy files from your local machine to your server (and vice versa), rather see the '` + os.Args[0] + ` scp' command.

EXAMPLE

  # Copy file "test.txt" from workspace root inside target "folder-a":
  ` + os.Args[0] + ` cp common-files/test.txt common-files/folder-a

  # Copy a file from a workspace to another:
  ` + os.Args[0] + ` cp common-files/test.txt personal-files/folder-b

  # Copy the full content of a folder inside another
  ` + os.Args[0] + ` cp common-files/test/* common-files/folder-c
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		target := args[1]

		var sourceNodes []string
		if path.Base(source) == "*" {
			nodes, err := rest.ListNodesPath(source)
			if err != nil {
				log.Println("could not list nodes path", err)
			}
			sourceNodes = nodes
		} else {
			sourceNodes = []string{source}
		}

		params := rest.CopyParams(sourceNodes, target)
		jobID, err := rest.CopyJob(params)
		if err != nil {
			log.Fatalln("could not run job:", err.Error())
		}

		err = rest.MonitorJob(jobID)
		if err != nil {
			log.Fatalln("could not monitor job", err.Error())
		}
	},
}

func init() {
	RootCmd.AddCommand(cpCmd)
}
