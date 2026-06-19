// embedset <pageId> <setId> — embed an existing Set/Collection inline as a
// dataview block (live board) at the top of a page. gRPC.
package main
import ("context";"fmt";"os"
 "github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service"
 "github.com/anyproto/anytype-heart/pkg/lib/pb/model";"github.com/anyproto/anytype-cli/core")
func main(){
 page,set:=os.Args[1],os.Args[2]
 err:=core.GRPCCall(func(ctx context.Context,c service.ClientCommandsClient)error{
  r,e:=c.BlockCreate(ctx,&pb.RpcBlockCreateRequest{ContextId:page,TargetId:"",Position:model.Block_Bottom,
   Block:&model.Block{Content:&model.BlockContentOfDataview{Dataview:&model.BlockContentDataview{
    TargetObjectId:set}}}})
  if e!=nil{return e}
  fmt.Printf("embedded set %s as block %s\n",set,r.BlockId)
  return nil
 })
 if err!=nil{fmt.Fprintln(os.Stderr,"ERROR:",err);os.Exit(1)}
}
