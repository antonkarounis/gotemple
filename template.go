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

var (
	BASE = "base.html"
)

type TemplateManagerOptions struct {
	RootPath    string
	IncludePath string
	funcMap     map[string]any
}

// -----------------------------------

type TemplateManager struct {
	storedTemplates map[string]*template.Template
	baseExists      bool
	options         TemplateManagerOptions
	funcMap         template.FuncMap
	watchedDirs     []string //list of paths being watched
}

func NewTemplateManager(options TemplateManagerOptions) (*TemplateManager, error) {
	// TODO: validate options
	// TODO: start filesystem watcher if configured

	if options.funcMap == nil {
		options.funcMap = make(map[string]any)
	}

	tm := TemplateManager{
		options:     options,
		funcMap:     options.funcMap,
		watchedDirs: make([]string, 0),
	}

	err := tm.reloadTemplates()
	if err != nil {
		return nil, err
	}
	return &tm, nil
}

func (tm *TemplateManager) reloadTemplates() error {
	tm.storedTemplates = make(map[string]*template.Template)

	// load includes
	absIncludePath, err := filepath.Abs(tm.options.IncludePath)
	if err != nil {
		return err
	}
	includes, err := template.New("root").ParseGlob(filepath.Join(absIncludePath, "*"))
	if err != nil {
		return err
	}

	base := includes.Lookup(BASE)
	if base != nil {
		tm.baseExists = true
	}

	// load templates
	absRootPath, err := filepath.Abs(tm.options.RootPath)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(absRootPath, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && err == nil && d.Type().IsRegular() {
			relativePath, err := filepath.Rel(absRootPath, path)
			if err != nil {
				return err
			}

			newTemplate, err := includes.Clone()
			if err != nil {
				return err
			}

			newTemplate, err = newTemplate.ParseFiles(path)
			if err != nil {
				return err
			}

			tm.storedTemplates[relativePath] = newTemplate
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (tm *TemplateManager) GetExecutor(templatePath string, exampleModel any) (*TemplateExecutor, error) {
	tmpl := tm.storedTemplates[templatePath]
	if tmpl == nil {
		return nil, errors.New("couldn't find template: " + templatePath)
	}

	var name string = ""
	if tm.baseExists {
		name = BASE
	} else {
		name = templatePath
	}
	err := govalidtemple.ValidateViewModel(exampleModel, tmpl, name)
	if err != nil {
		return nil, fmt.Errorf("couldn't validate view model for [%v]: %v", templatePath, err.Error())
	}

	return &TemplateExecutor{
		targetTemplate: tmpl,
		templateName:   templatePath,
		baseExists:     tm.baseExists,
	}, nil
}

func (tm *TemplateManager) GetTemplate(templatePath string, exampleModel any) (*template.Template, error) {
	tmpl := tm.storedTemplates[templatePath]
	if tmpl == nil {
		return nil, errors.New("couldn't find template: " + templatePath)
	}

	var name string = ""
	if tm.baseExists {
		name = BASE
	} else {
		name = templatePath
	}
	err := govalidtemple.ValidateViewModel(exampleModel, tmpl, name)
	if err != nil {
		return nil, fmt.Errorf("couldn't validate view model for [%v]: %v", templatePath, err.Error())
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
	templateName   string
	baseExists     bool
}

func (tm TemplateExecutor) ExecuteToString(data any) (string, error) {
	buffer := bytes.NewBufferString("")
	var err error = nil
	if tm.baseExists {
		err = tm.targetTemplate.ExecuteTemplate(buffer, BASE, data)
	} else {
		err = tm.targetTemplate.ExecuteTemplate(buffer, tm.templateName, data)
	}

	if err != nil {
		return "", fmt.Errorf("error executing template [%v]: %v", tm.targetTemplate.Name(), err.Error())
	}

	return buffer.String(), nil
}

func (tm TemplateExecutor) ExecuteToWriter(writer io.Writer, data any) error {
	var err error = nil
	if tm.baseExists {
		err = tm.targetTemplate.ExecuteTemplate(writer, BASE, data)
	} else {
		err = tm.targetTemplate.ExecuteTemplate(writer, tm.templateName, data)
	}

	if err != nil {
		return fmt.Errorf("error executing template [%v]: %v", tm.targetTemplate.Name(), err.Error())
	}

	return nil
}
