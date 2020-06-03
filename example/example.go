package example

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

//Serves templates with SSE js object (mainly for tests)
func MainHandler(w http.ResponseWriter, r *http.Request) {
	//Read template
	t, err := template.ParseFiles("example/templates/index.html")
	if err != nil {
		log.Panicf("Error parsing template: %s", err)
	}

	//Render template
	var env_vars = map[string]string{"port": os.Getenv("SSE_PORT"), "user": os.Getenv("SSE_CLIENT_USERNAME"), "password": os.Getenv("SSE_CLIENT_PASSWORD")}
	t.Execute(w, env_vars)
}
