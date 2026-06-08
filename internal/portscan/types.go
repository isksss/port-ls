package portscan

type Entry struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`
	Address   string `json:"address"`
	State     string `json:"state"`
	PID       *int   `json:"pid"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Query struct {
	All     bool
	Verbose bool
}

type Diagnostic struct {
	Provider   string
	Command    string
	ExitStatus string
	Stderr     string
}

type Provider interface {
	Name() string
	List(Query) ([]Entry, []Diagnostic, error)
}

type ProviderSet interface {
	List(Query) ([]Entry, []Diagnostic, error)
}

type ListOptions struct {
	All       bool
	Port      *int
	TCP       bool
	UDP       bool
	Name      string
	Address   string
	States    []string
	Host      bool
	Verbose   bool
	Namespace string
}

type FreeOptions struct {
	Start       *int
	End         *int
	UseTCP      bool
	UseUDP      bool
	Address     string
	JSON        bool
	Verbose     bool
	DefaultUsed bool
}
