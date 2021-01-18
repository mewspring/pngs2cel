package main

import (
	"html/template"
	"log"
	"os"
)

func main() {
	t, err := template.ParseGlob("*.tmpl")
	if err != nil {
		log.Fatalf("%+v", err)
	}
	tmpl := t.Lookup("copy.tmpl")
	for flareDir := 1; flareDir <= 8; flareDir++ {
		dir := (((flareDir + 2) - 1) % 8) + 1
		data := map[string]interface{}{
			"Dir":      dir,
			"FlareDir": flareDir,
		}
		if err := tmpl.Execute(os.Stdout, data); err != nil {
			log.Fatalf("%+v", err)
		}
	}
}
