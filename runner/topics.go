package main

// Every question topic in banks/*.json must be listed here; the area groups topics in the report.
var areas = []struct {
	Name   string
	Topics []string
}{
	{"Current affairs and GK", []string{"Current affairs", "Tech GK"}},
	{"Quantitative aptitude", []string{"Percentages", "Time and work", "Speed and distance", "Interest", "Averages", "Ratio",
		"Profit and loss", "Number system", "Algebra", "Mensuration", "Number series", "Mixtures", "Permutations", "Ages", "Clocks", "Calendar", "Probability"}},
	{"Reasoning", []string{"Coding-decoding", "Syllogisms", "Directions", "Blood relations", "Odd one out", "Ranking", "Analogy",
		"Letter series", "Puzzles", "Critical reasoning"}},
	{"English", []string{"Vocabulary", "Grammar", "Spelling", "Voice and speech", "Idioms", "Office terms", "Comprehension", "Sentence order"}},
	{"Computer fundamentals", []string{"Number systems", "Computer basics", "Computer architecture", "Memory and storage", "Hardware"}},
	{"C and data structures", []string{"C programming", "Data structures", "Algorithms"}},
	{"Operating systems and Linux", []string{"OS concepts", "Linux commands", "Users and permissions", "Linux system files", "Bash", "Disks and file systems", "Packages"}},
	{"Networks", []string{"Network basics", "Subnetting", "Network protocols", "Wi-Fi", "Network tools", "Sockets", "Cisco IOS"}},
	{"DBMS and MySQL", []string{"SQL", "DBMS theory", "MySQL admin"}},
	{"Tools and servers", []string{"Git", "LDAP", "Virtualization", "Servers", "Backup", "Cloud"}},
	{"Web", []string{"HTML and CSS", "JavaScript", "PHP", "Ajax"}},
	{"Python", []string{"Python", "Data visualization", "Django"}},
	{"Security", []string{"Security", "Firewall"}},
}

func areaOf(topic string) string {
	for _, a := range areas {
		for _, t := range a.Topics {
			if t == topic {
				return a.Name
			}
		}
	}
	return ""
}
