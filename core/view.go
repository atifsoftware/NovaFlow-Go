package core

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// LoadViews walks viewsDir for *.html templates (skipping the "layouts"
// subfolder) and parses each one together with every file in
// viewsDir/layouts, so every page can `{{define "content"}}` and rely on
// layouts/main.html's `{{define "layout"}}...{{template "content" .}}...{{end}}`.
//
// The returned map is keyed by the view's path relative to viewsDir with
// the extension stripped, e.g. "home/index" for views/home/index.html —
// this is the string passed to Context.View().
func LoadViews(viewsDir string) (map[string]*template.Template, error) {
	views := map[string]*template.Template{}
	layoutsDir := filepath.Join(viewsDir, "layouts")

	var layoutFiles []string
	_ = filepath.WalkDir(layoutsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".html") {
			layoutFiles = append(layoutFiles, path)
		}
		return nil
	})

	err := filepath.WalkDir(viewsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}
		if strings.HasPrefix(path, layoutsDir) {
			return nil
		}
		rel, _ := filepath.Rel(viewsDir, path)
		key := strings.TrimSuffix(rel, ".html")
		key = filepath.ToSlash(key)

		files := append([]string{path}, layoutFiles...)
		tpl, err := template.New(filepath.Base(path)).ParseFiles(files...)
		if err != nil {
			return err
		}
		views[key] = tpl
		return nil
	})
	if err != nil {
		return nil, err
	}
	return views, nil
}
