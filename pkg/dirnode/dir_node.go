package dirnode

type DirNode struct {
	FullPath string
	Name     string
}

// DirNode implements fmt.Stringer which charm.land/bubbles uses to render it in the tree bubble.
func (d *DirNode) String() string {
	return d.Name
}
