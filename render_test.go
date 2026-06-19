package main

import (
	"strings"
	"testing"

	"hubcap/internal/github"
)

func TestRenderIssueContentCmd_RendersBodyToMessage(t *testing.T) {
	issue := github.Issue{Body: "# Hello\n\nSome **bold** text."}
	pal := resolvePalette("")
	cmd := renderIssueContentCmd(issue, 80, pal)
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd")
	}
	msg := cmd()
	rendered, ok := msg.(issueContentRenderedMsg)
	if !ok {
		t.Fatalf("expected issueContentRenderedMsg, got %T", msg)
	}
	if rendered.content == "" {
		t.Error("expected non-empty rendered content for non-empty body")
	}
}

func TestRenderIssueContentCmd_EmptyBody_StillReturnsMessage(t *testing.T) {
	issue := github.Issue{Body: ""}
	pal := resolvePalette("")
	cmd := renderIssueContentCmd(issue, 80, pal)
	msg := cmd()
	if _, ok := msg.(issueContentRenderedMsg); !ok {
		t.Fatalf("expected issueContentRenderedMsg for empty body, got %T", msg)
	}
}

func TestRenderPRContentCmd_RendersBodyToMessage(t *testing.T) {
	pr := github.PullRequest{Body: "## Summary\n\nFixes the bug."}
	pal := resolvePalette("")
	cmd := renderPRContentCmd(pr, 80, pal)
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd")
	}
	msg := cmd()
	rendered, ok := msg.(prContentRenderedMsg)
	if !ok {
		t.Fatalf("expected prContentRenderedMsg, got %T", msg)
	}
	if rendered.content == "" {
		t.Error("expected non-empty rendered content for non-empty body")
	}
}

func TestRenderPRContentCmd_EmptyBody_StillReturnsMessage(t *testing.T) {
	pr := github.PullRequest{Body: ""}
	pal := resolvePalette("")
	cmd := renderPRContentCmd(pr, 80, pal)
	msg := cmd()
	if _, ok := msg.(prContentRenderedMsg); !ok {
		t.Fatalf("expected prContentRenderedMsg for empty body, got %T", msg)
	}
}

func TestRenderMarkdown_RepeatedCallsSameWidth_Consistent(t *testing.T) {
	body := "# Title\n\nParagraph with `code`."
	first := renderMarkdown(body, 80, "")
	second := renderMarkdown(body, 80, "")
	if first != second {
		t.Error("repeated renderMarkdown calls with same input should return identical output")
	}
	if first == "" {
		t.Error("expected non-empty output")
	}
}

func TestRenderMarkdown_BodyAppearsInOutput(t *testing.T) {
	body := "Hello unique-string-xyz"
	out := renderMarkdown(body, 80, "")
	if !strings.Contains(out, "unique-string-xyz") {
		t.Errorf("expected output to contain input text, got: %q", out)
	}
}
