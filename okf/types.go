package okf

// A Bundle is a parsed Open Knowledge Format bundle.
type Bundle struct {
	Root     string
	Version  string
	Concepts []Concept
	Invalid  []InvalidConcept
	Index    []ReservedFile
	Logs     []ReservedFile
	Metadata Metadata
}

// Metadata records OKF format metadata when it is known.
type Metadata struct {
	Version    string
	Format     string
	SourcePath string
}

// An InvalidConcept records a concept document whose frontmatter could not be
// parsed. [ParseBundle] keeps these rather than failing the whole bundle;
// [ValidateBundle] reports each as a conformance error.
type InvalidConcept struct {
	ID         string
	SourcePath string
	Err        error
}

// A Concept is one OKF concept document.
type Concept struct {
	ID          string
	Type        string
	Title       string
	Description string
	Resource    string
	Tags        []string
	Timestamp   string
	FrontMatter []Field
	Body        string
	Metadata    Metadata
}

// A Field is one frontmatter field.
type Field struct {
	Key    string
	Values []string
}

// A ReservedFile is a reserved OKF index.md or log.md file.
type ReservedFile struct {
	Name        string
	Body        string
	FrontMatter []Field
	Root        bool
	Metadata    Metadata
}
