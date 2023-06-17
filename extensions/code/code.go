package code

import (
	"html/template"

	"github.com/emad-elsaid/xlog"
	"github.com/yuin/goldmark/ast"
)

func init() {
	xlog.RegisterCommand(Commands)
	xlog.Post(`/\+/code`, PlayHandler)
}

type PlayCommand struct {
	page xlog.Page
}

func (p *PlayCommand) Icon() string          { return "fa-solid fa-play" }
func (p *PlayCommand) Name() string          { return "Run code" }
func (p *PlayCommand) Link() string          { return "https://emadelsaid.com" }
func (p *PlayCommand) OnClick() template.JS  { return "" }
func (p *PlayCommand) Widget() template.HTML { return "" }

func Commands(p xlog.Page) []xlog.Command {
	commands := []xlog.Command{
		&PlayCommand{
			page: p,
		},
	}

	if _, ok := xlog.FindInAST[*ast.CodeBlock](p.AST()); ok {
		return commands
	}

	if _, ok := xlog.FindInAST[*ast.FencedCodeBlock](p.AST()); ok {
		return commands
	}

	return []xlog.Command{}
}

func PlayHandler(w xlog.Response, r xlog.Request) xlog.Output {
	return xlog.Redirect("/")
}
