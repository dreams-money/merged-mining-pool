package rpc

func (r *RPCClient) Check() bool {
	_, err := r.GetBlockTemplate()
	if err != nil {
		return false
	}
	return true
}
