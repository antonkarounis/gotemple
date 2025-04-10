package basic

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/nullfocus/gotemple"
)

func main() {
	tm, _ := gotemple.NewTemplateManager(gotemple.TemplateManagerOptions{
		RootPath:    "./www/",
		IncludePath: "./include/",
	})

	r := chi.NewRouter()

	r.Get("/", tm.ExecutorRoute("index.html", IndexViewModel{}, indexController))
}

func indexController(w http.ResponseWriter, r *http.Request, te *gotemple.TemplateExecutor) {
	generatedPage, err := te.ExecuteToString(IndexViewModel{
		Name: "Bob Dobbs",
	})

	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, "500 error")
	}

	render.Status(r, http.StatusOK)
	render.HTML(w, r, generatedPage)
}

type IndexViewModel struct {
	Name string
}
