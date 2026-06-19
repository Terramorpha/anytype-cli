// hidegroup <context> <blockId> <viewId> <visibleGroupId>...
// Sets the Kanban group order for a dataview view: lists the given group ids
// as visible (in order) and hides the "empty" (no value) group. gRPC.
package main
import ("context";"fmt";"os"
 "github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service"
 "github.com/anyproto/anytype-heart/pkg/lib/pb/model";"github.com/anyproto/anytype-cli/core")
func main(){
 ctxId,block,view:=os.Args[1],os.Args[2],os.Args[3]; vis:=os.Args[4:]
 groups:=[]*model.BlockContentDataviewViewGroup{}
 for i,g:=range vis{ groups=append(groups,&model.BlockContentDataviewViewGroup{GroupId:g,Index:int32(i),Hidden:false}) }
 groups=append(groups,&model.BlockContentDataviewViewGroup{GroupId:"empty",Index:int32(len(vis)),Hidden:true})
 err:=core.GRPCCall(func(c0 context.Context,c service.ClientCommandsClient)error{
  _,e:=c.BlockDataviewGroupOrderUpdate(c0,&pb.RpcBlockDataviewGroupOrderUpdateRequest{
   ContextId:ctxId,BlockId:block,
   GroupOrder:&model.BlockContentDataviewGroupOrder{ViewId:view,ViewGroups:groups}})
  if e!=nil{return e}
  fmt.Printf("group order set: %d visible + empty hidden\n",len(vis))
  return nil
 })
 if err!=nil{fmt.Fprintln(os.Stderr,"ERROR:",err);os.Exit(1)}
}
