package main
import ("context";"fmt";"os";"github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service";"github.com/anyproto/anytype-cli/core")
func main(){
 _=core.GRPCCall(func(ctx context.Context,c service.ClientCommandsClient)error{
  _,e:=c.BlockListDelete(ctx,&pb.RpcBlockListDeleteRequest{ContextId:os.Args[1],BlockIds:os.Args[2:]})
  if e!=nil{return e}; fmt.Println("deleted"); return nil
 })
}
