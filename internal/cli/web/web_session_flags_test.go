package web

import (
	"context"
	"flag"
	"strings"
	"testing"

	webcore "github.com/rudrankriyam/App-Store-Connect-CLI/internal/web"
)

func TestBindWebSessionFlagsIncludesDeprecatedTwoFactorAlias(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags := bindWebSessionFlags(fs)

	if flags.twoFactorCode == nil {
		t.Fatal("expected deprecated two-factor-code pointer to be populated")
	}

	twoFactorCodeFlag := fs.Lookup(deprecatedTwoFactorCodeFlagName)
	if twoFactorCodeFlag == nil {
		t.Fatalf("expected --%s to be registered", deprecatedTwoFactorCodeFlagName)
		return
	}
	if !strings.Contains(twoFactorCodeFlag.Usage, "Deprecated:") {
		t.Fatalf("expected deprecated help text, got %q", twoFactorCodeFlag.Usage)
	}

	if fs.Lookup("two-factor-code-command") == nil {
		t.Fatal("expected --two-factor-code-command to remain registered")
	}
	if fs.Lookup("provider-id") == nil {
		t.Fatal("expected --provider-id to be registered")
	}
	if fs.Lookup("public-provider-id") == nil {
		t.Fatal("expected --public-provider-id to be registered")
	}
}

func TestResolveWebSessionForCommandPassesTwoFactorCodeCommand(t *testing.T) {
	restoreResolve := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode, twoFactorCodeCommand string) (*webcore.AuthSession, string, error) {
		if appleID != "user@example.com" {
			t.Fatalf("appleID = %q, want %q", appleID, "user@example.com")
		}
		if twoFactorCode != "" {
			t.Fatalf("twoFactorCode = %q, want empty", twoFactorCode)
		}
		if twoFactorCodeCommand != "osascript /tmp/get-apple-2fa-code.scpt" {
			t.Fatalf("twoFactorCodeCommand = %q, want osascript helper", twoFactorCodeCommand)
		}
		return &webcore.AuthSession{}, "test", nil
	})
	t.Cleanup(restoreResolve)

	flags := webSessionFlags{
		appleID:              ptrTo("user@example.com"),
		twoFactorCode:        ptrTo(""),
		twoFactorCodeCommand: ptrTo("osascript /tmp/get-apple-2fa-code.scpt"),
	}

	session, err := resolveWebSessionForCommand(context.Background(), flags)
	if err != nil {
		t.Fatalf("resolveWebSessionForCommand() error = %v", err)
	}
	if session == nil {
		t.Fatal("expected session")
	}
}

func TestResolveWebSessionForCommandSelectsProvider(t *testing.T) {
	expected := &webcore.AuthSession{UserEmail: "user@example.com"}
	restoreResolve := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode, twoFactorCodeCommand string) (*webcore.AuthSession, string, error) {
		return expected, "cache", nil
	})
	t.Cleanup(restoreResolve)

	origSelectProvider := selectWebProviderFn
	origPersist := persistWebSessionFn
	t.Cleanup(func() {
		selectWebProviderFn = origSelectProvider
		persistWebSessionFn = origPersist
	})

	selected := false
	selectWebProviderFn = func(ctx context.Context, session *webcore.AuthSession, selection webcore.ProviderSelection) error {
		selected = true
		if session != expected {
			t.Fatal("expected resolved session to be selected")
		}
		if selection.ProviderID != 123456 {
			t.Fatalf("ProviderID = %d, want 123456", selection.ProviderID)
		}
		if selection.PublicProviderID != "TEAM123" {
			t.Fatalf("PublicProviderID = %q, want TEAM123", selection.PublicProviderID)
		}
		session.ProviderID = selection.ProviderID
		session.PublicProviderID = selection.PublicProviderID
		return nil
	}
	persisted := false
	persistWebSessionFn = func(session *webcore.AuthSession) error {
		persisted = true
		if session != expected {
			t.Fatal("expected selected session to be persisted")
		}
		return nil
	}

	providerID := int64(123456)
	flags := webSessionFlags{
		appleID:              ptrTo("user@example.com"),
		twoFactorCode:        ptrTo(""),
		twoFactorCodeCommand: ptrTo(""),
		providerID:           &providerID,
		publicProviderID:     ptrTo("TEAM123"),
	}

	session, err := resolveWebSessionForCommand(context.Background(), flags)
	if err != nil {
		t.Fatalf("resolveWebSessionForCommand() error = %v", err)
	}
	if session != expected {
		t.Fatal("expected selected session")
	}
	if !selected {
		t.Fatal("expected provider selection")
	}
	if !persisted {
		t.Fatal("expected selected provider session to be persisted")
	}
}

func ptrTo(value string) *string {
	return &value
}
