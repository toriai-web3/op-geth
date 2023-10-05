package api

// API describes the set of methods offered over the RPC interface
type API struct {
	Svcname string      // service name
	Version string      // api version
	Service interface{} // api service instance which holds the methods
	Public  bool        // indication if the methods must be considered safe for public use
}
