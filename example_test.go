package gb_test

import (
	"log"
	"path/filepath"

	"github.com/constabulary/gb"
)

func ExampleNewProject() {

	// Every project begins with a project root.
	// Normally you'd check this out of source control.
	root := filepath.Join("home", "dfc", "devel", "demo")

	// Create a new Project passing in the source directories
	// under this project's root.
	proj := gb.NewProject(root)

	// Create a new Context from the Project. A Context holds
	// the state of a specific compilation or test within the Project.
	ctx, err := gb.NewContext(proj)
	if err != nil {
		log.Fatal("Could not create new context:", err)
	}

	// Always remember to clean up your Context
	ctx.Destroy()
}
