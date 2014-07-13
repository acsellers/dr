package parse

import "testing"

func TestIndexSingle(t *testing.T) {
	expected := `var _ = doc.RegisterIndexes(
User{},
[]string{"Name"},
)`
	result, err := ParseIndexes([]string{"index(", "Name User", ")"})
	if err != nil {
		t.Fatal("Received error:", err)
	}
	if result != expected {
		t.Fatalf("Expected:\n%s\n\nRecieved:\n%s", expected, result)
	}
}

func TestIndexTwoSingle(t *testing.T) {
	expected := `var _ = doc.RegisterIndexes(
User{},
[]string{"Name"},
[]string{"Password"},
)`
	result, err := ParseIndexes([]string{"index(", "Name User", "Password", ")"})
	if err != nil {
		t.Fatal("Received error:", err)
	}
	if result != expected {
		t.Fatalf("Expected:\n%s\n\nRecieved:\n%s", expected, result)
	}
}

func TestIndexMulti(t *testing.T) {
	expected := `var _ = doc.RegisterIndexes(
User{},
[]string{"Name", "Login"},
)`
	result, err := ParseIndexes([]string{"index(", "Name, Login User", ")"})
	if err != nil {
		t.Fatal("Received error:", err)
	}
	if result != expected {
		t.Fatalf("Expected:\n%s\n\nRecieved:\n%s", expected, result)
	}
}

func TestIndexTwoMulti(t *testing.T) {
	expected := `var _ = doc.RegisterIndexes(
User{},
[]string{"Name", "Login"},
[]string{"Login", "APIKey"},
)`
	result, err := ParseIndexes([]string{"index(", "Name, Login User", "Login, APIKey", ")"})
	if err != nil {
		t.Fatal("Received error:", err)
	}
	if result != expected {
		t.Fatalf("Expected:\n%s\n\nRecieved:\n%s", expected, result)
	}
}

func TestIndexComment(t *testing.T) {
	expected := `var _ = doc.RegisterIndexes(
// Do not delete, important for reports
User{},
[]string{"Name"},
)`
	result, err := ParseIndexes([]string{"index(", "// Do not delete, important for reports", "Name User", ")"})
	if err != nil {
		t.Fatal("Received error:", err)
	}
	if result != expected {
		t.Fatalf("Expected:\n%s\n\nRecieved:\n%s", expected, result)
	}
}
