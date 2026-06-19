// embedset <pageId> <setId> — embed an existing Set/Collection inline as a LIVE
// dataview block: create an empty dataview block, then populate it from the set
// via BlockDataviewCreateFromExistingObject (copies its views). gRPC.
package main
import ("context";"fmt";"os"
 "github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service"
 "github.com/anyproto/anytype-heart/pkg/lib/pb/model";"github.com/anyproto/anytype-cli/core")
func main(){
 page,set:=os.Args[1],os.Args[2]
 err:=core.GRPCCall(func(ctx context.Context,c service.ClientCommandsClient)error{
  r,e:=c.BlockCreate(ctx,&pb.RpcBlockCreateRequest{ContextId:page,Position:model.Block_Bottom,
   Block:&model.Block{Content:&model.BlockContentOfDataview{Dataview:&model.BlockContentDataview{}}}})
  if e!=nil{return fmt.Errorf("create block: %w",e)}
  bid:=r.BlockId
  resp,e:=c.BlockDataviewCreateFromExistingObject(ctx,&pb.RpcBlockDataviewCreateFromExistingObjectRequest{
   ContextId:page,BlockId:bid,TargetObjectId:set})
  if e!=nil{return fmt.Errorf("populate: %w",e)}
  if resp.Error!=nil && resp.Error.Code!=pb.RpcBlockDataviewCreateFromExistingObjectResponseError_NULL{
   return fmt.Errorf("%s: %s",resp.Error.Code,resp.Error.Description)
  }
  fmt.Printf("inline set embedded: block %s -> %s (views copied)\n",bid,set)
  return nil
 })
 if err!=nil{fmt.Fprintln(os.Stderr,"ERROR:",err);os.Exit(1)}
}
