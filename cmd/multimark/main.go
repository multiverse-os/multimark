package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	markdown "../../markdown"
	// TODO: This is a bad API
	html "../../renderer/html"
)

func main() {
	fmt.Println("[markdown] Multi(verse) Markdown")
	fmt.Println("================================")
	fmt.Println("A specialized whitelisting, markdown specifically for use")
	fmt.Println("with Multiverse OS applications and features")

	if len(os.Args) != 2 {
		fmt.Println("[ERROR] Missing arguments, must supply original MD file and")
		fmt.Println("the resulting HML file.")
		PrintHelp()
	}

	markdownFile, err := ioutil.ReadFile(os.Args[0])
	if err != nil {
		fmt.Println("[ERROR] Failed to load specific markdown file, does it exist?")
		fmt.Println("and is it a valid file?")
		PrintHelp()
		os.Exit(1)
	} else {
		fmt.Println("Successfully loaded markdown file:")
		fmt.Print(string(markdownFile))
	}

	source := []byte("a  \r\nb\n")
	var b bytes.Buffer
	// TODO: "New" is not really the right expressive name
	// TODO: Kinda hate this method of declaration, its overly confusing.
	if err := markdown.New(markdown.WithRendererOptions(html.WithXHTML())).Convert(source, &b); err != nil {
		fmt.Println("[ERROR] Invalid markdown! Try using a linter if the above text appears to be correct markdown")
		PrintHelp()
		os.Exit(1)
	} else {
		fmt.Println("Successfully converted markdown file to HTML:")
		fmt.Println(b.String())
		fmt.Println("Attempting to write to specified file:", os.Args[1])

		err := ioutil.WriteFile(args[1], b.String(), 0644)
		if err != nil {
			fmt.Println("[ERROR] Failed to write HTML file!")
			PrintHelp()
			os.Exit(1)
		} else {
			fmt.Println("Successfully wrote HTML file based on our markdown file!")
			os.Exit(0)
		}
	}
}

func PrintHelp() {
	fmt.Println("Usage:\n")
	fmt.Println("  multimark research.notes.md article.html\n")
}
