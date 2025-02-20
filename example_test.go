package arg

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"
)

func split(s string) []string {
	return strings.Split(s, " ")
}

// This example demonstrates basic usage
func Example() {
	// These are the args you would pass in on the command line
	os.Args = split("./example --foo=hello --bar")

	var args struct {
		Foo string
		Bar bool
	}
	MustParse(&args)
	fmt.Println(args.Foo, args.Bar)
	// output: hello true
}

// This example demonstrates arguments that have default values
func Example_defaultValues() {
	// These are the args you would pass in on the command line
	os.Args = split("./example")

	var args struct {
		Foo string `default:"abc"`
	}
	MustParse(&args)
	fmt.Println(args.Foo)
	// output: abc
}

// This example demonstrates arguments that are required
func Example_requiredArguments() {
	// These are the args you would pass in on the command line
	os.Args = split("./example --foo=abc --bar")

	var args struct {
		Foo string `arg:"required"`
		Bar bool
	}
	MustParse(&args)
	fmt.Println(args.Foo, args.Bar)
	// output: abc true
}

// This example demonstrates positional arguments
func Example_positionalArguments() {
	// These are the args you would pass in on the command line
	os.Args = split("./example in out1 out2 out3")

	var args struct {
		Input  string   `arg:"positional"`
		Output []string `arg:"positional"`
	}
	MustParse(&args)
	fmt.Println("In:", args.Input)
	fmt.Println("Out:", args.Output)
	// output:
	// In: in
	// Out: [out1 out2 out3]
}

// This example demonstrates arguments that have multiple values
func Example_multipleValues() {
	// The args you would pass in on the command line
	os.Args = split("./example --database localhost --ids 1 2 3")

	var args struct {
		Database string
		IDs      []int64
	}
	MustParse(&args)
	fmt.Printf("Fetching the following IDs from %s: %v", args.Database, args.IDs)
	// output: Fetching the following IDs from localhost: [1 2 3]
}

// This example demonstrates arguments with keys and values
func Example_mappings() {
	// The args you would pass in on the command line
	os.Args = split("./example --userids john=123 mary=456")

	var args struct {
		UserIDs map[string]int
	}
	MustParse(&args)
	fmt.Println(args.UserIDs)
	// output: map[john:123 mary:456]
}

type commaSeparated struct {
	M map[string]string
}

func (c *commaSeparated) UnmarshalText(b []byte) error {
	c.M = make(map[string]string)
	for _, part := range strings.Split(string(b), ",") {
		pos := strings.Index(part, "=")
		if pos == -1 {
			return fmt.Errorf("error parsing %q, expected format key=value", part)
		}
		c.M[part[:pos]] = part[pos+1:]
	}
	return nil
}

// This example demonstrates arguments with keys and values separated by commas
func Example_mappingWithCommas() {
	// The args you would pass in on the command line
	os.Args = split("./example --values one=two,three=four")

	var args struct {
		Values commaSeparated
	}
	MustParse(&args)
	fmt.Println(args.Values.M)
	// output: map[one:two three:four]
}

// This eample demonstrates multiple value arguments that can be mixed with
// other arguments.
func Example_multipleMixed() {
	os.Args = split("./example -c cmd1 db1 -f file1 db2 -c cmd2 -f file2 -f file3 db3 -c cmd3")
	var args struct {
		Commands  []string `arg:"-c,separate"`
		Files     []string `arg:"-f,separate"`
		Databases []string `arg:"positional"`
	}
	MustParse(&args)
	fmt.Println("Commands:", args.Commands)
	fmt.Println("Files:", args.Files)
	fmt.Println("Databases:", args.Databases)

	// output:
	// Commands: [cmd1 cmd2 cmd3]
	// Files: [file1 file2 file3]
	// Databases: [db1 db2 db3]
}

// This example shows the usage string generated by go-arg
func Example_helpText() {
	// These are the args you would pass in on the command line
	os.Args = split("./example --help")

	var args struct {
		Input    string   `arg:"positional"`
		Output   []string `arg:"positional"`
		Verbose  bool     `arg:"-v" help:"verbosity level"`
		Dataset  string   `help:"dataset to use"`
		Optimize int      `arg:"-O,--optim" help:"optimization level"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stdout = os.Stdout

	MustParse(&args)

	// output:
	// Usage: example [--verbose] [--dataset DATASET] [--optim OPTIM] INPUT [OUTPUT [OUTPUT ...]]
	//
	// Positional arguments:
	//   INPUT
	//   OUTPUT
	//
	// Options:
	//   --verbose, -v          verbosity level
	//   --dataset DATASET      dataset to use
	//   --optim OPTIM, -O OPTIM
	//                          optimization level
	//   --help, -h             display this help and exit
}

// This example shows the usage string generated by go-arg with customized placeholders
func Example_helpPlaceholder() {
	// These are the args you would pass in on the command line
	os.Args = split("./example --help")

	var args struct {
		Input    string   `arg:"positional" placeholder:"SRC"`
		Output   []string `arg:"positional" placeholder:"DST"`
		Optimize int      `arg:"-O" help:"optimization level" placeholder:"LEVEL"`
		MaxJobs  int      `arg:"-j" help:"maximum number of simultaneous jobs" placeholder:"N"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stdout = os.Stdout

	MustParse(&args)

	// output:

	// Usage: example [--optimize LEVEL] [--maxjobs N] SRC [DST [DST ...]]

	// Positional arguments:
	//   SRC
	//   DST

	// Options:
	//   --optimize LEVEL, -O LEVEL
	//                          optimization level
	//   --maxjobs N, -j N      maximum number of simultaneous jobs
	//   --help, -h             display this help and exit
}

// This example shows the usage string generated by go-arg when using subcommands
func Example_helpTextWithSubcommand() {
	// These are the args you would pass in on the command line
	os.Args = split("./example --help")

	type getCmd struct {
		Item string `arg:"positional" help:"item to fetch"`
	}

	type listCmd struct {
		Format string `help:"output format"`
		Limit  int
	}

	var args struct {
		Verbose bool
		Get     *getCmd  `arg:"subcommand" help:"fetch an item and print it"`
		List    *listCmd `arg:"subcommand" help:"list available items"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stdout = os.Stdout

	MustParse(&args)

	// output:
	// Usage: example [--verbose] <command> [<args>]
	//
	// Options:
	//   --verbose
	//   --help, -h             display this help and exit
	//
	// Commands:
	//   get                    fetch an item and print it
	//   list                   list available items
}

// This example shows the usage string generated by go-arg when using subcommands
func Example_helpTextWhenUsingSubcommand() {
	// These are the args you would pass in on the command line
	os.Args = split("./example get --help")

	type getCmd struct {
		Item string `arg:"positional" help:"item to fetch"`
	}

	type listCmd struct {
		Format string `help:"output format"`
		Limit  int
	}

	var args struct {
		Verbose bool
		Get     *getCmd  `arg:"subcommand" help:"fetch an item and print it"`
		List    *listCmd `arg:"subcommand" help:"list available items"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stdout = os.Stdout

	MustParse(&args)

	// output:
	// Usage: example get ITEM
	//
	// Positional arguments:
	//   ITEM                   item to fetch
	//
	// Global options:
	//   --verbose
	//   --help, -h             display this help and exit
}

// This example shows how to print help for an explicit subcommand
func Example_writeHelpForSubcommand() {
	// These are the args you would pass in on the command line
	os.Args = split("./example get --help")

	type getCmd struct {
		Item string `arg:"positional" help:"item to fetch"`
	}

	type listCmd struct {
		Format string `help:"output format"`
		Limit  int
	}

	var args struct {
		Verbose bool
		Get     *getCmd  `arg:"subcommand" help:"fetch an item and print it"`
		List    *listCmd `arg:"subcommand" help:"list available items"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stdout = os.Stdout

	p, err := NewParser(Config{}, &args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = p.WriteHelpForSubcommand(os.Stdout, "list")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// output:
	// Usage: example list [--format FORMAT] [--limit LIMIT]
	//
	// Options:
	//   --format FORMAT        output format
	//   --limit LIMIT
	//
	// Global options:
	//   --verbose
	//   --help, -h             display this help and exit
}

// This example shows how to print help for a subcommand that is nested several levels deep
func Example_writeHelpForSubcommandNested() {
	// These are the args you would pass in on the command line
	os.Args = split("./example get --help")

	type mostNestedCmd struct {
		Item string
	}

	type nestedCmd struct {
		MostNested *mostNestedCmd `arg:"subcommand"`
	}

	type topLevelCmd struct {
		Nested *nestedCmd `arg:"subcommand"`
	}

	var args struct {
		TopLevel *topLevelCmd `arg:"subcommand"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stdout = os.Stdout

	p, err := NewParser(Config{}, &args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = p.WriteHelpForSubcommand(os.Stdout, "toplevel", "nested", "mostnested")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// output:
	// Usage: example toplevel nested mostnested [--item ITEM]
	//
	// Options:
	//   --item ITEM
	//   --help, -h             display this help and exit
}

// This example shows the error string generated by go-arg when an invalid option is provided
func Example_errorText() {
	// These are the args you would pass in on the command line
	os.Args = split("./example --optimize INVALID")

	var args struct {
		Input    string   `arg:"positional"`
		Output   []string `arg:"positional"`
		Verbose  bool     `arg:"-v" help:"verbosity level"`
		Dataset  string   `help:"dataset to use"`
		Optimize int      `arg:"-O,help:optimization level"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stderr = os.Stdout

	MustParse(&args)

	// output:
	// Usage: example [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] INPUT [OUTPUT [OUTPUT ...]]
	// error: error processing --optimize: strconv.ParseInt: parsing "INVALID": invalid syntax
}

// This example shows the error string generated by go-arg when an invalid option is provided
func Example_errorTextForSubcommand() {
	// These are the args you would pass in on the command line
	os.Args = split("./example get --count INVALID")

	type getCmd struct {
		Count int
	}

	var args struct {
		Get *getCmd `arg:"subcommand"`
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stderr = os.Stdout

	MustParse(&args)

	// output:
	// Usage: example get [--count COUNT]
	// error: error processing --count: strconv.ParseInt: parsing "INVALID": invalid syntax
}

// This example demonstrates use of subcommands
func Example_subcommand() {
	// These are the args you would pass in on the command line
	os.Args = split("./example commit -a -m what-this-commit-is-about")

	type CheckoutCmd struct {
		Branch string `arg:"positional"`
		Track  bool   `arg:"-t"`
	}
	type CommitCmd struct {
		All     bool   `arg:"-a"`
		Message string `arg:"-m"`
	}
	type PushCmd struct {
		Remote      string `arg:"positional"`
		Branch      string `arg:"positional"`
		SetUpstream bool   `arg:"-u"`
	}
	var args struct {
		Checkout *CheckoutCmd `arg:"subcommand:checkout"`
		Commit   *CommitCmd   `arg:"subcommand:commit"`
		Push     *PushCmd     `arg:"subcommand:push"`
		Quiet    bool         `arg:"-q"` // this flag is global to all subcommands
	}

	// This is only necessary when running inside golang's runnable example harness
	osExit = func(int) {}
	stderr = os.Stdout

	MustParse(&args)

	switch {
	case args.Checkout != nil:
		fmt.Printf("checkout requested for branch %s\n", args.Checkout.Branch)
	case args.Commit != nil:
		fmt.Printf("commit requested with message \"%s\"\n", args.Commit.Message)
	case args.Push != nil:
		fmt.Printf("push requested from %s to %s\n", args.Push.Branch, args.Push.Remote)
	}

	// output:
	// commit requested with message "what-this-commit-is-about"
}

func Example_allSupportedTypes() {
	// These are the args you would pass in on the command line
	os.Args = []string{}

	var args struct {
		Bool     bool
		Byte     byte
		Rune     rune
		Int      int
		Int8     int8
		Int16    int16
		Int32    int32
		Int64    int64
		Float32  float32
		Float64  float64
		String   string
		Duration time.Duration
		URL      url.URL
		Email    mail.Address
		MAC      net.HardwareAddr
	}

	// go-arg supports each of the types above, as well as pointers to any of
	// the above and slices of any of the above. It also supports any types that
	// implements encoding.TextUnmarshaler.

	MustParse(&args)

	// output:
}
