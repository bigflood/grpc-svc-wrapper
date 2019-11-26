package pb

import "google.golang.org/grpc"

func GetHelloServiceDesc() *grpc.ServiceDesc {
	return &_Hello_serviceDesc
}
