package cmd

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var (
	updateToVersion string
	updateDryRun    bool
	devChannel      bool
	// unstableChannel bool
	defaultChannel = common.UpdateStableChannel
)

var updateBinCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for available updates and apply them",
	Long: `
DESCRIPTION	
	
  Without argument, the 'update' command lists available updates.
  To apply the actual update, re-run the command specifying the target version with the --version flag.

  By default, we check for update in the stable channel, that is: we only install binaries that have been properly released.
  If necessary, you can use the --dev flag to switch **at your own risks** to the pre-release channel.
`,
	Run: func(cmd *cobra.Command, args []string) {

		// if set it will use the selected channel to list and perform the update
		if devChannel {
			defaultChannel = common.UpdateDevChannel
		}

		binaries, e := rest.LoadUpdates(context.Background(), defaultChannel)
		if e != nil {
			log.Fatal("Cannot retrieve available updates", e)
		}
		if len(binaries) == 0 {
			c := color.New(color.FgRed)
			c.Println("\nNo updates are available for this version")
			c.Println("")
			return
		}

		if updateToVersion == "" {
			// List versions
			c := color.New(color.FgGreen)
			c.Println("\nNew packages are available. Please run the following command to upgrade to a given version")
			c.Println("")
			c = color.New(color.FgBlack, color.Bold)
			c.Println(os.Args[0] + " update --version=x.y.z")
			c.Println("")

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Version", "UpdatePackage Name", "Description"})

			for _, bin := range binaries {
				table.Append([]string{bin.Version, bin.Label, bin.Description})
			}

			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.Render()

		} else {

			var apply *rest.UpdatePackage
			for _, binary := range binaries {
				if binary.Version == updateToVersion {
					apply = binary
				}
			}
			if apply == nil {
				log.Fatal("Cannot find the requested version")
			}

			c := color.New(color.FgBlack)
			fmt.Println("Updating binary now")
			c.Println("")
			pgChan := make(chan float64)
			errorChan := make(chan error)
			doneChan := make(chan bool)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case pg := <-pgChan:
						fmt.Printf("\rDownloading binary: %v%%", math.Floor(pg*100))
					case e := <-errorChan:
						fmt.Printf("\rError while updating binary: " + e.Error())
						return
					case <-doneChan:
						// TODO use another color or let the default color
						fmt.Printf("\n\nCells Client binary successfully upgraded\n")
						return
					}
				}
			}()
			rest.ApplyUpdate(context.Background(), apply, updateDryRun, pgChan, doneChan, errorChan)
			wg.Wait()
		}

	},
}

func init() {
	updateBinCmd.Flags().StringVarP(&updateToVersion, "version", "v", "", "Specify the version to be installed and trigger the actual upgrade")
	updateBinCmd.Flags().BoolVarP(&updateDryRun, "dry-run", "d", false, "If set, this flag will grab the package and save it to the tmp directory instead of replacing current binary")
	updateBinCmd.Flags().BoolVar(&devChannel, "dev", false, "If set this flag will use the dev channel to load the updates")

	RootCmd.AddCommand(updateBinCmd)
}
