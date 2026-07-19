package goplus

import "cmd/vet/testdata/goplus/model"

func load() (model.Decision, error) { return model.Decision.Allow{}, nil }

func save() error { return nil }

func inspect(decision model.Decision) (string, error) {
	try loaded := load()
	try save()
	defer {
		_ = loaded.Variant()
	}
	qualified := model.Decision.Deny{Reason: "no"}
	_ = qualified
	switch loaded {
	case Allow:
		return "yes", nil
	case Deny:
		return loaded.Reason, nil
	case nil:
		return "", nil
	}
	panic("unreachable")
}
