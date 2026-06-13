package project

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// --- helm ---

func TestHelm_SetVersion_OnlyVersionLine(t *testing.T) {
	dir := t.TempDir()
	in := "apiVersion: v2\nname: app\nversion: 0.1.0\nappVersion: \"1.0\"\n"
	p := writeFile(t, dir, helmFile, in)

	changed, err := (helm{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "apiVersion: v2\nname: app\nversion: 0.2.0\nappVersion: \"1.0\"\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestHelm_SetVersion_PreservesQuotes(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, helmFile, "version: \"0.1.0\"\n")

	if _, err := (helm{}).SetVersion(p, "1.2.4-rc.1"); err != nil {
		t.Fatal(err)
	}
	if got, want := readFile(t, p), "version: \"1.2.4-rc.1\"\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestHelm_SetVersion_PreservesTrailingComment(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, helmFile, "version: 0.1.0 # chart version\n")

	if _, err := (helm{}).SetVersion(p, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if got, want := readFile(t, p), "version: 0.2.0 # chart version\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestHelm_SetVersion_MissingField_NoChange(t *testing.T) {
	dir := t.TempDir()
	in := "apiVersion: v2\nname: app\nappVersion: 1.0\n"
	p := writeFile(t, dir, helmFile, in)

	changed, err := (helm{}).SetVersion(p, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
	if got := readFile(t, p); got != in {
		t.Fatalf("file mutated: %q", got)
	}
}

func TestHelm_SetVersion_DoesNotTouchAppVersion(t *testing.T) {
	dir := t.TempDir()
	in := "appVersion: 0.1.0\n"
	p := writeFile(t, dir, helmFile, in)

	changed, _ := (helm{}).SetVersion(p, "9.9.9")
	if changed || readFile(t, p) != in {
		t.Fatalf("appVersion should be untouched, changed=%v content=%q", changed, readFile(t, p))
	}
}

func TestHelm_Detect(t *testing.T) {
	dir := t.TempDir()
	if _, ok := (helm{}).Detect(dir); ok {
		t.Fatal("should not detect in empty dir")
	}
	writeFile(t, dir, helmFile, "version: 0.1.0\n")
	rel, ok := (helm{}).Detect(dir)
	if !ok || rel != helmFile {
		t.Fatalf("rel=%q ok=%v", rel, ok)
	}
}

// --- npm ---

func TestNpm_SetVersion_ByteIdenticalExceptValue(t *testing.T) {
	dir := t.TempDir()
	in := "{\n  \"name\": \"app\",\n  \"version\": \"0.0.0\",\n  \"private\": true\n}\n"
	p := writeFile(t, dir, npmFile, in)

	changed, err := (npm{}).SetVersion(p, "0.1.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "{\n  \"name\": \"app\",\n  \"version\": \"0.1.0\",\n  \"private\": true\n}\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestNpm_SetVersion_MissingField_NoChange(t *testing.T) {
	dir := t.TempDir()
	in := "{\n  \"name\": \"app\"\n}\n"
	p := writeFile(t, dir, npmFile, in)

	changed, err := (npm{}).SetVersion(p, "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if changed || readFile(t, p) != in {
		t.Fatalf("changed=%v content=%q", changed, readFile(t, p))
	}
}

// --- cargo ---

func TestCargo_SetVersion_Basic(t *testing.T) {
	dir := t.TempDir()
	in := "[package]\nname = \"app\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[dependencies]\nserde = \"1.0\"\n"
	p := writeFile(t, dir, cargoFile, in)

	changed, err := (cargo{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "[package]\nname = \"app\"\nversion = \"0.2.0\"\nedition = \"2021\"\n\n[dependencies]\nserde = \"1.0\"\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCargo_SetVersion_DoesNotTouchDependencies(t *testing.T) {
	dir := t.TempDir()
	in := "[package]\nname = \"app\"\nversion = \"0.1.0\"\n\n[dependencies.foo]\nversion = \"1.0\"\n"
	p := writeFile(t, dir, cargoFile, in)

	changed, err := (cargo{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "[package]\nname = \"app\"\nversion = \"0.2.0\"\n\n[dependencies.foo]\nversion = \"1.0\"\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCargo_SetVersion_MissingField_NoChange(t *testing.T) {
	dir := t.TempDir()
	in := "[package]\nname = \"app\"\nedition = \"2021\"\n"
	p := writeFile(t, dir, cargoFile, in)

	changed, err := (cargo{}).SetVersion(p, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
	if got := readFile(t, p); got != in {
		t.Fatalf("file mutated: %q", got)
	}
}

func TestCargo_Detect(t *testing.T) {
	dir := t.TempDir()
	if _, ok := (cargo{}).Detect(dir); ok {
		t.Fatal("should not detect in empty dir")
	}
	writeFile(t, dir, cargoFile, "[package]\nversion = \"0.1.0\"\n")
	rel, ok := (cargo{}).Detect(dir)
	if !ok || rel != cargoFile {
		t.Fatalf("rel=%q ok=%v", rel, ok)
	}
}

// --- pom ---

func TestPom_SetVersion_Basic(t *testing.T) {
	dir := t.TempDir()
	in := "<?xml version=\"1.0\"?>\n<project>\n  <artifactId>app</artifactId>\n  <version>0.1.0</version>\n</project>\n"
	p := writeFile(t, dir, pomFile, in)

	changed, err := (pom{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "<?xml version=\"1.0\"?>\n<project>\n  <artifactId>app</artifactId>\n  <version>0.2.0</version>\n</project>\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPom_SetVersion_DoesNotTouchDependencyVersion(t *testing.T) {
	dir := t.TempDir()
	in := "<project>\n  <version>0.1.0</version>\n  <dependencies>\n    <dependency>\n      <version>1.0.0</version>\n    </dependency>\n  </dependencies>\n</project>\n"
	p := writeFile(t, dir, pomFile, in)

	changed, err := (pom{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "<project>\n  <version>0.2.0</version>\n  <dependencies>\n    <dependency>\n      <version>1.0.0</version>\n    </dependency>\n  </dependencies>\n</project>\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPom_SetVersion_DoesNotTouchParentVersion(t *testing.T) {
	dir := t.TempDir()
	in := "<project>\n  <parent>\n    <version>3.0.0</version>\n  </parent>\n  <version>0.1.0</version>\n</project>\n"
	p := writeFile(t, dir, pomFile, in)

	changed, err := (pom{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "<project>\n  <parent>\n    <version>3.0.0</version>\n  </parent>\n  <version>0.2.0</version>\n</project>\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPom_SetVersion_MissingField_NoChange(t *testing.T) {
	dir := t.TempDir()
	in := "<project>\n  <artifactId>app</artifactId>\n</project>\n"
	p := writeFile(t, dir, pomFile, in)

	changed, err := (pom{}).SetVersion(p, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
	if got := readFile(t, p); got != in {
		t.Fatalf("file mutated: %q", got)
	}
}

func TestPom_Detect(t *testing.T) {
	dir := t.TempDir()
	if _, ok := (pom{}).Detect(dir); ok {
		t.Fatal("should not detect in empty dir")
	}
	writeFile(t, dir, pomFile, "<project><version>0.1.0</version></project>\n")
	rel, ok := (pom{}).Detect(dir)
	if !ok || rel != pomFile {
		t.Fatalf("rel=%q ok=%v", rel, ok)
	}
}

// --- pyproject ---

func TestPyproject_SetVersion_Basic(t *testing.T) {
	dir := t.TempDir()
	in := "[project]\nname = \"app\"\nversion = \"0.1.0\"\nrequires-python = \">=3.11\"\n"
	p := writeFile(t, dir, pyprojectFile, in)

	changed, err := (pyproject{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "[project]\nname = \"app\"\nversion = \"0.2.0\"\nrequires-python = \">=3.11\"\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPyproject_SetVersion_DoesNotTouchOtherSections(t *testing.T) {
	dir := t.TempDir()
	in := "[project]\nname = \"app\"\nversion = \"0.1.0\"\n\n[tool.foo]\nversion = \"9.9.9\"\n"
	p := writeFile(t, dir, pyprojectFile, in)

	changed, err := (pyproject{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "[project]\nname = \"app\"\nversion = \"0.2.0\"\n\n[tool.foo]\nversion = \"9.9.9\"\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPyproject_SetVersion_MissingField_NoChange(t *testing.T) {
	dir := t.TempDir()
	in := "[project]\nname = \"app\"\ndynamic = [\"version\"]\n"
	p := writeFile(t, dir, pyprojectFile, in)

	changed, err := (pyproject{}).SetVersion(p, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
	if got := readFile(t, p); got != in {
		t.Fatalf("file mutated: %q", got)
	}
}

func TestPyproject_Detect(t *testing.T) {
	dir := t.TempDir()
	if _, ok := (pyproject{}).Detect(dir); ok {
		t.Fatal("should not detect in empty dir")
	}
	writeFile(t, dir, pyprojectFile, "[project]\nversion = \"0.1.0\"\n")
	rel, ok := (pyproject{}).Detect(dir)
	if !ok || rel != pyprojectFile {
		t.Fatalf("rel=%q ok=%v", rel, ok)
	}
}

// --- dotnet ---

const dotnetTestFile = "app.csproj"

func TestDotnet_SetVersion_Basic(t *testing.T) {
	dir := t.TempDir()
	in := "<Project Sdk=\"Microsoft.NET.Sdk\">\n  <PropertyGroup>\n    <TargetFramework>net8.0</TargetFramework>\n    <Version>0.1.0</Version>\n  </PropertyGroup>\n</Project>\n"
	p := writeFile(t, dir, dotnetTestFile, in)

	changed, err := (dotnet{}).SetVersion(p, "0.2.0")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	want := "<Project Sdk=\"Microsoft.NET.Sdk\">\n  <PropertyGroup>\n    <TargetFramework>net8.0</TargetFramework>\n    <Version>0.2.0</Version>\n  </PropertyGroup>\n</Project>\n"
	if got := readFile(t, p); got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestDotnet_SetVersion_MissingField_NoChange(t *testing.T) {
	dir := t.TempDir()
	in := "<Project Sdk=\"Microsoft.NET.Sdk\">\n  <PropertyGroup>\n    <TargetFramework>net8.0</TargetFramework>\n  </PropertyGroup>\n</Project>\n"
	p := writeFile(t, dir, dotnetTestFile, in)

	changed, err := (dotnet{}).SetVersion(p, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
	if got := readFile(t, p); got != in {
		t.Fatalf("file mutated: %q", got)
	}
}

func TestDotnet_SetVersion_DoesNotTouchAssemblyVersion(t *testing.T) {
	dir := t.TempDir()
	in := "<Project>\n  <PropertyGroup>\n    <AssemblyVersion>0.1.0.0</AssemblyVersion>\n  </PropertyGroup>\n</Project>\n"
	p := writeFile(t, dir, dotnetTestFile, in)

	changed, _ := (dotnet{}).SetVersion(p, "9.9.9")
	if changed || readFile(t, p) != in {
		t.Fatalf("AssemblyVersion should be untouched, changed=%v content=%q", changed, readFile(t, p))
	}
}

func TestDotnet_Detect(t *testing.T) {
	dir := t.TempDir()
	if _, ok := (dotnet{}).Detect(dir); ok {
		t.Fatal("should not detect in empty dir")
	}
	writeFile(t, dir, dotnetTestFile, "<Project/>\n")
	rel, ok := (dotnet{}).Detect(dir)
	if !ok || rel != dotnetTestFile {
		t.Fatalf("rel=%q ok=%v", rel, ok)
	}
}

// --- registry ---

func TestDetect_Registry(t *testing.T) {
	dir := t.TempDir()
	if got := Detect(dir); len(got) != 0 {
		t.Fatalf("plain dir: %v", got)
	}
	writeFile(t, dir, helmFile, "version: 0.1.0\n")
	writeFile(t, dir, npmFile, "{\n  \"version\": \"0.1.0\"\n}\n")
	writeFile(t, dir, cargoFile, "[package]\nversion = \"0.1.0\"\n")
	writeFile(t, dir, dotnetTestFile, "<Project>\n  <PropertyGroup>\n    <Version>0.1.0</Version>\n  </PropertyGroup>\n</Project>\n")
	writeFile(t, dir, pyprojectFile, "[project]\nversion = \"0.1.0\"\n")
	writeFile(t, dir, pomFile, "<project><version>0.1.0</version></project>\n")

	got := Detect(dir)
	slices.Sort(got)
	want := []string{cargoFile, dotnetTestFile, helmFile, npmFile, pomFile, pyprojectFile}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPatch_AllDetected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, helmFile, "version: 0.1.0\n")
	writeFile(t, dir, npmFile, "{\n  \"version\": \"0.1.0\"\n}\n")
	writeFile(t, dir, cargoFile, "[package]\nversion = \"0.1.0\"\n")
	writeFile(t, dir, dotnetTestFile, "<Project>\n  <PropertyGroup>\n    <Version>0.1.0</Version>\n  </PropertyGroup>\n</Project>\n")
	writeFile(t, dir, pyprojectFile, "[project]\nversion = \"0.1.0\"\n")
	writeFile(t, dir, pomFile, "<project><version>0.1.0</version></project>\n")

	changed, err := Patch(dir, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(changed)
	want := []string{cargoFile, dotnetTestFile, helmFile, npmFile, pomFile, pyprojectFile}
	slices.Sort(want)
	if !slices.Equal(changed, want) {
		t.Fatalf("changed=%v want %v", changed, want)
	}
	if got := readFile(t, filepath.Join(dir, helmFile)); got != "version: 0.2.0\n" {
		t.Fatalf("helm: %q", got)
	}
	if got := readFile(t, filepath.Join(dir, cargoFile)); got != "[package]\nversion = \"0.2.0\"\n" {
		t.Fatalf("cargo: %q", got)
	}
	wantDotnet := "<Project>\n  <PropertyGroup>\n    <Version>0.2.0</Version>\n  </PropertyGroup>\n</Project>\n"
	if got := readFile(t, filepath.Join(dir, dotnetTestFile)); got != wantDotnet {
		t.Fatalf("dotnet: %q", got)
	}
	if got := readFile(t, filepath.Join(dir, pyprojectFile)); got != "[project]\nversion = \"0.2.0\"\n" {
		t.Fatalf("pyproject: %q", got)
	}
	if got := readFile(t, filepath.Join(dir, pomFile)); got != "<project><version>0.2.0</version></project>\n" {
		t.Fatalf("pom: %q", got)
	}
}

func TestPatch_NoProject_NothingChanged(t *testing.T) {
	dir := t.TempDir()
	changed, err := Patch(dir, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("changed=%v", changed)
	}
}

func TestPatch_DetectedButNoField_NotChangedNotError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, helmFile, "name: app\n") // no version key

	changed, err := Patch(dir, "0.2.0")
	if err != nil {
		t.Fatalf("missing field must not error: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("changed=%v", changed)
	}
}
