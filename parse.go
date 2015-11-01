package arg

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// spec represents a command line option
type spec struct {
	dest       reflect.Value
	long       string
	short      string
	multiple   bool
	required   bool
	positional bool
	help       string
	wasPresent bool
}

// MustParse processes command line arguments and exits upon failure.
func MustParse(dest ...interface{}) {
	err := Parse(dest...)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Parse processes command line arguments and stores the result in args.
func Parse(dest ...interface{}) error {
	return ParseFrom(os.Args[1:], dest...)
}

// ParseFrom processes command line arguments and stores the result in args.
func ParseFrom(args []string, dest ...interface{}) error {
	// Add the help option if one is not already defined
	var internal struct {
		Help bool `arg:"-h,help:print this help message"`
	}

	// Parse the spec
	dest = append(dest, &internal)
	spec, err := extractSpec(dest...)
	if err != nil {
		return err
	}

	// Process args
	err = processArgs(spec, args)
	if err != nil {
		return err
	}

	// If -h or --help were specified then print help
	if internal.Help {
		writeUsage(os.Stdout, spec)
		os.Exit(0)
	}

	// Validate
	return validate(spec)
}

// extractSpec gets specifications for each argument from the tags in a struct
func extractSpec(dests ...interface{}) ([]*spec, error) {
	var specs []*spec
	for _, dest := range dests {
		v := reflect.ValueOf(dest)
		if v.Kind() != reflect.Ptr {
			panic(fmt.Sprintf("%s is not a pointer (did you forget an ampersand?)", v.Type()))
		}
		v = v.Elem()
		if v.Kind() != reflect.Struct {
			panic(fmt.Sprintf("%T is not a struct pointer", dest))
		}

		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			// Check for the ignore switch in the tag
			field := t.Field(i)
			tag := field.Tag.Get("arg")
			if tag == "-" {
				continue
			}

			spec := spec{
				long: strings.ToLower(field.Name),
				dest: v.Field(i),
			}

			// Get the scalar type for this field
			scalarType := field.Type
			if scalarType.Kind() == reflect.Slice {
				spec.multiple = true
				scalarType = scalarType.Elem()
				if scalarType.Kind() == reflect.Ptr {
					scalarType = scalarType.Elem()
				}
			}

			// Check for unsupported types
			switch scalarType.Kind() {
			case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
				reflect.Map, reflect.Ptr, reflect.Struct,
				reflect.Complex64, reflect.Complex128:
				return nil, fmt.Errorf("%s.%s: %s fields are not supported", t.Name(), field.Name, scalarType.Kind())
			}

			// Look at the tag
			if tag != "" {
				for _, key := range strings.Split(tag, ",") {
					var value string
					if pos := strings.Index(key, ":"); pos != -1 {
						value = key[pos+1:]
						key = key[:pos]
					}

					switch {
					case strings.HasPrefix(key, "--"):
						spec.long = key[2:]
					case strings.HasPrefix(key, "-"):
						if len(key) != 2 {
							return nil, fmt.Errorf("%s.%s: short arguments must be one character only", t.Name(), field.Name)
						}
						spec.short = key[1:]
					case key == "required":
						spec.required = true
					case key == "positional":
						spec.positional = true
					case key == "help":
						spec.help = value
					default:
						return nil, fmt.Errorf("unrecognized tag '%s' on field %s", key, tag)
					}
				}
			}
			specs = append(specs, &spec)
		}
	}
	return specs, nil
}

// processArgs processes arguments using a pre-constructed spec
func processArgs(specs []*spec, args []string) error {
	// construct a map from --option to spec
	optionMap := make(map[string]*spec)
	for _, spec := range specs {
		if spec.positional {
			continue
		}
		if spec.long != "" {
			optionMap[spec.long] = spec
		}
		if spec.short != "" {
			optionMap[spec.short] = spec
		}
	}

	// process each string from the command line
	var allpositional bool
	var positionals []string

	// must use explicit for loop, not range, because we manipulate i inside the loop
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			allpositional = true
			continue
		}

		if !strings.HasPrefix(arg, "-") || allpositional {
			positionals = append(positionals, arg)
			continue
		}

		// check for an equals sign, as in "--foo=bar"
		var value string
		opt := strings.TrimLeft(arg, "-")
		if pos := strings.Index(opt, "="); pos != -1 {
			value = opt[pos+1:]
			opt = opt[:pos]
		}

		// lookup the spec for this option
		spec, ok := optionMap[opt]
		if !ok {
			return fmt.Errorf("unknown argument %s", arg)
		}
		spec.wasPresent = true

		// deal with the case of multiple values
		if spec.multiple {
			var values []string
			if value == "" {
				for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					values = append(values, args[i+1])
					i++
				}
			} else {
				values = append(values, value)
			}
			err := setSlice(spec.dest, values)
			if err != nil {
				return fmt.Errorf("error processing %s: %v", arg, err)
			}
			continue
		}

		// if it's a flag and it has no value then set the value to true
		if spec.dest.Kind() == reflect.Bool && value == "" {
			value = "true"
		}

		// if we have something like "--foo" then the value is the next argument
		if value == "" {
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "-") {
				return fmt.Errorf("missing value for %s", arg)
			}
			value = args[i+1]
			i++
		}

		err := setScalar(spec.dest, value)
		if err != nil {
			return fmt.Errorf("error processing %s: %v", arg, err)
		}
	}

	// process positionals
	for _, spec := range specs {
		if spec.positional {
			if spec.multiple {
				err := setSlice(spec.dest, positionals)
				if err != nil {
					return fmt.Errorf("error processing %s: %v", spec.long, err)
				}
				positionals = nil
			} else if len(positionals) > 0 {
				err := setScalar(spec.dest, positionals[0])
				if err != nil {
					return fmt.Errorf("error processing %s: %v", spec.long, err)
				}
				positionals = positionals[1:]
			} else if spec.required {
				return fmt.Errorf("%s is required", spec.long)
			}
		}
	}
	if len(positionals) > 0 {
		return fmt.Errorf("too many positional arguments at '%s'", positionals[0])
	}
	return nil
}

// validate an argument spec after arguments have been parse
func validate(spec []*spec) error {
	for _, arg := range spec {
		if !arg.positional && arg.required && !arg.wasPresent {
			return fmt.Errorf("--%s is required", arg.long)
		}
	}
	return nil
}

// parse a value as the apropriate type and store it in the struct
func setSlice(dest reflect.Value, values []string) error {
	if !dest.CanSet() {
		return fmt.Errorf("field is not writable")
	}

	var ptr bool
	elem := dest.Type().Elem()
	if elem.Kind() == reflect.Ptr {
		ptr = true
		elem = elem.Elem()
	}

	for _, s := range values {
		v := reflect.New(elem)
		if err := setScalar(v.Elem(), s); err != nil {
			return err
		}
		if ptr {
			v = v.Addr()
		}
		dest.Set(reflect.Append(dest, v.Elem()))
	}
	return nil
}

// set a value from a string
func setScalar(v reflect.Value, s string) error {
	if !v.CanSet() {
		return fmt.Errorf("field is not exported")
	}

	switch v.Kind() {
	case reflect.String:
		v.Set(reflect.ValueOf(s))
	case reflect.Bool:
		x, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x).Convert(v.Type()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x).Convert(v.Type()))
	case reflect.Float32, reflect.Float64:
		x, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(x).Convert(v.Type()))
	default:
		return fmt.Errorf("not a scalar type: %s", v.Kind())
	}
	return nil
}