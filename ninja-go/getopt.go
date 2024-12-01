package ninja_go

/* macros defined by this include file */
const (
	no_argument       = 0
	required_argument = 1
	OPTIONAL_ARG      = 2
)

type GETOPT_LONG_OPTION_T struct {
	name    string /* the name of the long option */
	has_arg int    /* one of the above macros */
	flag    *int   /* determines if getopt_long() returns a
	 * value for a long option; if it is
	 * non-NULL, 0 is returned as a function
	 * value and the value of val is stored in
	 * the area pointed to by flag.  Otherwise,
	 * val is returned. */
	val int /* determines the value to return if flag is
	 * NULL. */
}

type option GETOPT_LONG_OPTION_T

var optind int = 0

// var optarg string = ""
var opterr int = 1
var optopt int = '?'
