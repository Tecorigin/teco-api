package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
)

func TestReplaceByRulesCaseInsensitiveLiteral(t *testing.T) {
	changed, hits, out := replaceByRules("hello AAAA world", []setting.KeywordReplaceRule{{
		From: "aaaa",
		To:   "bbbb",
	}})

	if !changed {
		t.Fatalf("expected replacement to happen")
	}
	if len(hits) != 1 || hits[0] != "aaaa" {
		t.Fatalf("unexpected hits: %#v", hits)
	}
	if out != "hello bbbb world" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReplaceByRulesWildcard(t *testing.T) {
	changed, _, out := replaceByRules("start fooXYZbar end", []setting.KeywordReplaceRule{{
		From: "foo*bar",
		To:   "MATCH",
	}})

	if !changed {
		t.Fatalf("expected wildcard replacement to happen")
	}
	if out != "start MATCH end" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestReplaceByRulesRegex(t *testing.T) {
	changed, _, out := replaceByRules("ID: A123 ID: b456", []setting.KeywordReplaceRule{{
		From: `re:[ab]\d{3}`,
		To:   "X",
	}})

	if !changed {
		t.Fatalf("expected regex replacement to happen")
	}
	if out != "ID: X ID: X" {
		t.Fatalf("unexpected output: %s", out)
	}
}
