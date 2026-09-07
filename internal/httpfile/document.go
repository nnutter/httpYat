package httpfile

// Document is a parsed httpYac-style .http file focused on REST/JSON requests.
type Document struct {
	Path     string
	Globals  []Variable
	Requests []Request
}

// Request is a single HTTP request region.
type Request struct {
	Name        string
	Description string
	Method      string
	URL         string
	Version     string
	Headers     []Header
	Body        string
	Vars        []Variable
	Line        int

	explicitName bool
}

// MetaName is the # @name value, used with httpyac send -n.
func (r Request) MetaName() string {
	if r.explicitName {
		return r.Name
	}
	return ""
}

// Header is a single HTTP header field.
type Header struct {
	Name  string
	Value string
}

// Variable is a named substitution value.
type Variable struct {
	Name  string
	Value string
	Line  int
}

// Prepared is a request after variable substitution.
type Prepared struct {
	Name    string
	Method  string
	URL     string
	Headers []Header
	Body    string
}

// DisplayName returns the label shown in the request list.
func (r Request) DisplayName() string {
	if r.Name != "" {
		return r.Name
	}
	if r.Method == "" {
		return r.URL
	}
	if r.URL == "" {
		return r.Method
	}
	return r.Method + " " + r.URL
}
