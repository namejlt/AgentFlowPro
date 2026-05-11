package datasource

type ExecuteInput struct {
	GlobalVars map[string]any // workflow + manual merged
	Params     map[string]any // resolved per params_schema
}

type ExecuteResult struct {
	Raw      []byte
	Extracted string
	FromCache bool
}
