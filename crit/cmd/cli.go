package main

import (
	"log"

	"github.com/checkpoint-restore/go-criu/v5/crit"
	"github.com/spf13/cobra"
)

// The crit service used to invoke all commands
var c crit.CritSvc

// All members needed for crit struct
var inputFilePath, outputFilePath, inputDirPath, exploreType string
var pretty, noPayload bool

var rootCmd = &cobra.Command{
	Use:   "crit",
	Short: "A tool to manipulate CRIU image files",
	Long: `CRIU Image Tool (CRIT) is a command line tool to investigate
binary image files generated by CRIU and view them in JSON.
This is a Go implementation of the original Python app.
Find the complete documentation is at https://criu.org/CRIT`,
}

var decodeCmd = &cobra.Command{
	Use:   "decode",
	Short: "Convert binary image to JSON",
	Long: `Convert the input binary image to JSON and writes it to a file.
If no output file is provided, the JSON is printed to stdout.`,
	Run: func(cmd *cobra.Command, args []string) {
		c = crit.New(inputFilePath, outputFilePath, inputDirPath, exploreType, pretty, noPayload)
		if err := c.Decode(); err != nil {
			log.Fatal(err)
		}
	},
}

var encodeCmd = &cobra.Command{
	Use:   "encode",
	Short: "Convert JSON to binary file",
	Long:  "Convert the input JSON to a CRIU image file.",
	Run: func(cmd *cobra.Command, args []string) {
		c = crit.New(inputFilePath, outputFilePath, inputDirPath, exploreType, pretty, noPayload)
		if err := c.Encode(); err != nil {
			log.Fatal(err)
		}
	},
}

var showCmd = &cobra.Command{
	Use:   "show INPATH",
	Short: "Convert binary image to human-readable JSON",
	Long:  "Convert the input binary image to human-readable JSON and print to stdout",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFilePath = args[0]
		pretty = true
		c = crit.New(inputFilePath, outputFilePath, inputDirPath, exploreType, pretty, noPayload)
		if err := c.Decode(); err != nil {
			log.Fatal(err)
		}
	},
}

var infoCmd = &cobra.Command{
	Use:   "info INPATH",
	Short: "Show information about the image file",
	Long:  "Show information about the image file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFilePath = args[0]
		c = crit.New(inputFilePath, outputFilePath, inputDirPath, exploreType, pretty, noPayload)
		if err := c.Info(); err != nil {
			log.Fatal(err)
		}
	},
}

var xCmd = &cobra.Command{
	Use:   "x DIR {ps|fds|mems|rss}",
	Short: "Explore the image directory",
	Long:  "Explore the image directory with one of (ps, fds, mems, rss) options",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		inputDirPath = args[0]
		exploreType = args[1]
		c = crit.New(inputFilePath, outputFilePath, inputDirPath, exploreType, pretty, noPayload)
		if err := c.X(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	// Disable completion generation
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	// Decode options
	decodeCmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "Path to the CRIU binary file")
	decodeCmd.MarkFlagRequired("input")
	decodeCmd.Flags().StringVarP(&outputFilePath, "output", "o", "", "Path to the destination JSON file")
	decodeCmd.Flags().BoolVar(&pretty, "pretty", false, "Provide indented and multi-line JSON output")
	rootCmd.AddCommand(decodeCmd)
	// Encode options
	encodeCmd.Flags().StringVarP(&inputFilePath, "input", "i", "", "Path to the JSON file")
	encodeCmd.Flags().StringVarP(&outputFilePath, "output", "o", "", "Path to the destination image file")
	rootCmd.AddCommand(encodeCmd)
	// Show options
	showCmd.Flags().BoolVar(&noPayload, "nopl", false, "Do not show payload contents")
	rootCmd.AddCommand(showCmd)
	// Info and X commands
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(xCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
