package gtkutils

import (
	"errors"
	"log"

	"github.com/gotk3/gotk3/gtk"
)

func GetListStore(b *gtk.Builder, id string) (listStore *gtk.ListStore) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("List store error: %s", e)
		return nil
	}

	listStore, _ = obj.(*gtk.ListStore)
	return
}

func GetTreeSelection(b *gtk.Builder, id string) (treeSelection *gtk.TreeSelection) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Tree Selection error: %s", e)
		return nil
	}

	treeSelection, _ = obj.(*gtk.TreeSelection)
	return
}

func GetButton(b *gtk.Builder, id string) (btn *gtk.Button) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Button error: %s", e)
		return nil
	}

	btn, _ = obj.(*gtk.Button)
	return
}

func GetToggleButton(b *gtk.Builder, id string) (btn *gtk.ToggleButton) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Toggle button error: %s", e)
		return nil
	}

	btn, _ = obj.(*gtk.ToggleButton)
	return
}

func GetTreeView(b *gtk.Builder, id string) (treeView *gtk.TreeView) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Tree view error: %s", e)
		return nil
	}

	treeView, _ = obj.(*gtk.TreeView)
	return
}

func GetTreeViewColumn(b *gtk.Builder, id string) (treeViewColumn *gtk.TreeViewColumn) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Tree view error: %s", e)
		return nil
	}

	treeViewColumn, _ = obj.(*gtk.TreeViewColumn)
	return
}

func GetLabel(b *gtk.Builder, id string) (treeView *gtk.Label) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Label error: %s", e)
		return nil
	}

	treeView, _ = obj.(*gtk.Label)
	return
}

func GetScrolledWindow(b *gtk.Builder, id string) (treeView *gtk.ScrolledWindow) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Scrolled window error: %s", e)
		return nil
	}

	treeView, _ = obj.(*gtk.ScrolledWindow)
	return
}

func GetSpinner(b *gtk.Builder, id string) (treeView *gtk.Spinner) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Spinner error: %s", e)
		return nil
	}

	treeView, _ = obj.(*gtk.Spinner)
	return
}

func GetEntry(b *gtk.Builder, id string) (treeView *gtk.Entry) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Entry error: %s", e)
		return nil
	}

	treeView, _ = obj.(*gtk.Entry)
	return
}

func GetSearchEntry(b *gtk.Builder, id string) (searchEntry *gtk.SearchEntry) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Search Entry error: %s", e)
		return nil
	}

	searchEntry, _ = obj.(*gtk.SearchEntry)
	return
}

func GetComboBox(b *gtk.Builder, id string) (combobox *gtk.ComboBox) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("ComboBox error: %s", e)
		return nil
	}

	combobox, _ = obj.(*gtk.ComboBox)
	return
}

func GetCheckButton(b *gtk.Builder, id string) (el *gtk.CheckButton) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("CheckButton error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.CheckButton)
	return
}

func GetImage(b *gtk.Builder, id string) (el *gtk.Image) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Image error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.Image)
	return
}

func GetMenuItem(b *gtk.Builder, id string) (el *gtk.MenuItem) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("MenuItem error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.MenuItem)
	return
}

func GetCheckMenuItem(b *gtk.Builder, id string) (el *gtk.CheckMenuItem) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("CheckMenuItem error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.CheckMenuItem)
	return
}

func GetBox(b *gtk.Builder, id string) (el *gtk.Box) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Box error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.Box)
	return
}

func GetSeparator(b *gtk.Builder, id string) (el *gtk.Separator) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Separator error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.Separator)
	return
}

func GetToolButton(b *gtk.Builder, id string) (el *gtk.ToolButton) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Tool Button error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.ToolButton)
	return
}

func GetToggleToolButton(b *gtk.Builder, id string) (el *gtk.ToggleToolButton) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("ToggleToolButton error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.ToggleToolButton)
	return
}

func GetNotebook(b *gtk.Builder, id string) (el *gtk.Notebook) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("Notebook error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.Notebook)
	return
}

func GetCellRendererText(b *gtk.Builder, id string) (el *gtk.CellRendererText) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("CellRendererText error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.CellRendererText)
	return
}

func GetTextBuffer(b *gtk.Builder, id string) (el *gtk.TextBuffer) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("TextBuffer error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.TextBuffer)
	return
}

func GetTextView(b *gtk.Builder, id string) (el *gtk.TextView) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("TextView error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.TextView)
	return
}

func GetTreeModelFilter(b *gtk.Builder, id string) (el *gtk.TreeModelFilter) {
	obj, e := b.GetObject(id)
	if e != nil {
		log.Printf("TreeModelFilter error: %s", e)
		return nil
	}

	el, _ = obj.(*gtk.TreeModelFilter)
	return
}

func GetFilterValues(entryKeyword *gtk.Entry, cmbBoxRepo *gtk.ComboBox, cmbBoxLang *gtk.ComboBox,
	chckBtnInstalled *gtk.CheckButton) (keywordP, repoP, langP *string, onlyInstalled bool) {
	var e error

	keyword, e := entryKeyword.GetText()
	if e != nil {
		log.Fatalf("Error: %s", e)
	}
	if keyword != "" {
		keywordP = &keyword
	}

	repo := cmbBoxRepo.GetActiveID()
	if repo != "" {
		repoP = &repo
	}

	lang := cmbBoxLang.GetActiveID()
	if lang != "" {
		langP = &lang
	}

	onlyInstalled = chckBtnInstalled.GetActive()

	return
}

func FindFirstIterInTreeSelection(ls *gtk.ListStore, s *gtk.TreeSelection) (*gtk.TreeIter, error) {
	rows := s.GetSelectedRows(ls)
	if rows.Length() < 1 {
		return nil, errors.New("No selected elements")
	}

	path := rows.Data().(*gtk.TreePath)
	iter, e := ls.GetIter(path)

	return iter, e
}

func GetIterFromTextPathInListStore(ls *gtk.ListStore, path string) (*gtk.TreeIter, error) {
	treePath, e := gtk.TreePathNewFromString(path)
	if e != nil {
		return nil, e
	}

	iter, e := ls.GetIter(treePath)
	return iter, e
}
