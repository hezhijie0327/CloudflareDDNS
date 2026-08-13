package main

import "strings"

func banner(version string) string {
	const b = `
███████╗     ██╗██████╗ ██████╗ ███╗   ██╗███████╗
╚══███╔╝     ██║██╔══██╗██╔══██╗████╗  ██║██╔════╝
  ███╔╝      ██║██║  ██║██║  ██║██╔██╗ ██║███████╗
 ███╔╝  ██   ██║██║  ██║██║  ██║██║╚██╗██║╚════██║
███████╗╚█████╔╝██████╔╝██████╔╝██║ ╚████║███████║
╚══════╝ ╚════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚══════╝
  {version}

  Lightweight zero-dependency DDNS updater
  https://github.com/hezhijie0327/ZJDDNS
__________________________________\o/_______
`

	// Plain substitution, NOT fmt.Sprintf: the artwork is inert text and any
	// literal '%' added to it later would otherwise be re-interpreted as a
	// format verb.
	return strings.ReplaceAll(b[1:], "{version}", version)
}
