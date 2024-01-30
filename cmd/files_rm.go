package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/rest"
)

var (
	force        bool
	wildcardChar = "%"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Trash files or folders",
	Long: `
DESCRIPTION
	
  Delete specified files or folders. 
	
  In fact, it only moves specified files or folders to the recycle bin 
  that is at the root of the corresponding workspace, the trashed objects 
  can be restored (from the web UI, this feature is not yet implemented 
  in the Cells Client) 

EXAMPLES

  # Generic example:
  ` + os.Args[0] + ` rm <workspace-slug>/path/to/resource

  # Remove a single file:
  ` + os.Args[0] + ` rm common-files/target.txt

  # Remove recursively inside a folder, the wildcard is '%':
  ` + os.Args[0] + ` rm common-files/folder/%

  # Remove a folder and all its children (even if it is not empty)
  ` + os.Args[0] + ` rm common-files/folder

  # Remove multiple files
  ` + os.Args[0] + ` rm common-files/file-1.txt common-files/file-2.txt

  # You can force the deletion with the '--force' flag (to avoid the Yes or No)
  ` + os.Args[0] + ` rm -f common-files/file-1.txt
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Ask for user approval before deleting
		p := promptui.Select{Label: "Are you sure", Items: []string{"No", "Yes"}}
		if !force {
			if _, resp, e := p.Run(); resp == "No" && e == nil {
				log.Println("Nothing will be deleted")
				return
			}
		}
		ctx := cmd.Context()
		targetNodes := make([]string, 0)
		for _, arg := range args {
			_, exists := rest.StatNode(ctx, strings.TrimRight(arg, wildcardChar))
			if !exists {
				log.Printf("Node not found %v, could not delete\n", arg)
			}
			if path.Base(arg) == wildcardChar {
				dir, _ := path.Split(arg)
				newArg := path.Join(dir, "*")
				nodes, err := rest.ListNodesPath(ctx, newArg)

				// Remove recycle_bin from targetedNodes
				for i, c := range nodes {
					if path.Base(c) == "recycle_bin" {
						nodes = append(nodes[:i], nodes[i+1:]...)
					}
				}

				if err != nil {
					log.Fatalf("Could not list nodes inside %s, aborting. Cause: %s\n", path.Dir(arg), err.Error())
				}
				targetNodes = append(targetNodes, nodes...)
			} else {
				targetNodes = append(targetNodes, arg)
			}
		}

		if len(targetNodes) <= 0 {
			log.Println("Nothing to delete")
			return
		}

		jobUUID, err := rest.DeleteNode(ctx, targetNodes)
		if err != nil {
			log.Fatalf("could not delete nodes, cause: %s\n", err)
		}

		var wg sync.WaitGroup
		for _, id := range jobUUID {
			wg.Add(1)
			go func(id string) {
				err := rest.MonitorJob(ctx, id)
				defer wg.Done()
				if err != nil {
					log.Printf("could not monitor job, %s\n", id)
				}
			}(id)
		}
		wg.Wait()

		fmt.Println("Nodes have been moved to the Recycle Bin")
	},
}

func init() {
	RootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "Do not ask for user approval")
}
