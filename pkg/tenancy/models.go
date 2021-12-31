package tenancy

type Tenant struct {
	Id          string
	Scheme      string
	Name        string
	Host        string
	VirtualPath string
}
