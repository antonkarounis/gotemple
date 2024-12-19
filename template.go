package gotemple

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/nullfocus/govalidtemple"
)

type TemplateManagerOptions struct {
	RootPath        string
	IncludePath     string
	WatchForChanges bool
}

// -----------------------------------

type TemplateManager struct {
	rootTemplate *template.Template
}

func NewTemplateManager(options TemplateManagerOptions) (*TemplateManager, error) {
	rootTemplate := template.New("root")

	// TODO: validate options

	// TODO: start filesystem watcher if configured
	// for includes, just need to reload changed ones, filename will overwrite in the root
	// for templates, need to reparse and then add to root with the same name, overwriting existing

	// load includes
	absIncludePath, err := filepath.Abs(options.IncludePath)
	fmt.Println("loading includes:")
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(absIncludePath, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && err == nil && d.Type().IsRegular() {
			fmt.Println("  " + d.Name())
			//insert each include into the root
			rootTemplate, err = rootTemplate.ParseFiles(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// load templates
	absRootPath, err := filepath.Abs(options.RootPath)
	fmt.Println("loading templates:")
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(absRootPath, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && err == nil && d.Type().IsRegular() {
			//parse into new template
			newTemplate, err := template.ParseFiles(path)
			if err != nil {
				return err
			}
			relativePath, err := filepath.Rel(absRootPath, path)
			if err != nil {
				return err
			}
			fmt.Println("  " + relativePath)
			//merge into root
			rootTemplate, err = rootTemplate.AddParseTree(relativePath, newTemplate.Tree)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	fmt.Println()

	return &TemplateManager{
		rootTemplate: rootTemplate,
	}, nil
}

func (tm *TemplateManager) GetExecutor(templatePath string, exampleModel any) (*TemplateExecutor, error) {
	tmpl := tm.rootTemplate.Lookup(templatePath)
	if tmpl == nil {
		return nil, errors.New("couldn't find template: " + templatePath)
	}

	err := govalidtemple.ValidateViewModel(exampleModel, tmpl, templatePath)
	if err != nil {
		return nil, errors.New("couldn't validate view model: " + err.Error())
	}

	return &TemplateExecutor{
		targetTemplate: tmpl,
	}, nil
}

func (tm *TemplateManager) GetTemplate(templatePath string, exampleModel any) (*template.Template, error) {
	tmpl := tm.rootTemplate.Lookup(templatePath)
	if tmpl == nil {
		return nil, errors.New("couldn't find template: " + templatePath)
	}

	err := govalidtemple.ValidateViewModel(exampleModel, tmpl, templatePath)
	if err != nil {
		return nil, errors.New("couldn't validate view model: " + err.Error())
	}

	return tmpl, nil
}

func (tm *TemplateManager) TemplateRoute(
	templatePath string,
	exampleModel any,
	fn func(w http.ResponseWriter, r *http.Request, tmpl *template.Template),
) func(w http.ResponseWriter, r *http.Request) {
	tmpl, err := tm.GetTemplate(templatePath, exampleModel)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, tmpl)
	}
}

func (tm *TemplateManager) ExecutorRoute(
	templatePath string,
	exampleModel any,
	fn func(w http.ResponseWriter, r *http.Request, te *TemplateExecutor),
) func(w http.ResponseWriter, r *http.Request) {
	executor, err := tm.GetExecutor(templatePath, exampleModel)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, executor)
	}
}

// -----------------------------------

type TemplateExecutor struct {
	targetTemplate *template.Template
}

func (tm TemplateExecutor) ExecuteToString(data any) (string, error) {
	buffer := bytes.NewBufferString("")
	err := tm.targetTemplate.Execute(buffer, data)
	if err != nil {
		return "", fmt.Errorf("error executing template [%v]: %v", tm.targetTemplate.Name(), err.Error())
	}

	return buffer.String(), nil
}

func (tm TemplateExecutor) ExecuteToWriter(writer io.Writer, data any) error {
	err := tm.targetTemplate.Execute(writer, data)
	if err != nil {
		return fmt.Errorf("error executing template [%v]: %v", tm.targetTemplate.Name(), err.Error())
	}

	return nil
}
