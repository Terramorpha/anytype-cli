// pagedeck <pageId> <coverId> <objId>...  — set a gradient cover and append
// link-CARD blocks (clickable object cards) to a page. gRPC.
package main
import ("context";"fmt";"os"
 "github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service"
 "github.com/anyproto/anytype-heart/pkg/lib/pb/model";"github.com/gogo/protobuf/types";"github.com/anyproto/anytype-cli/core")
func main(){
 page,cover:=os.Args[1],os.Args[2]; objs:=os.Args[3:]
 err:=core.GRPCCall(func(ctx context.Context,c service.ClientCommandsClient)error{
  // cover (gradient)
  if _,e:=c.ObjectSetDetails(ctx,&pb.RpcObjectSetDetailsRequest{ContextId:page,Details:[]*model.Detail{
   {Key:"coverType",Value:&types.Value{Kind:&types.Value_NumberValue{NumberValue:3}}},
   {Key:"coverId",Value:&types.Value{Kind:&types.Value_StringValue{StringValue:cover}}},
  }});e!=nil{return e}
  fmt.Println("cover set")
  prev:=""
  for _,o:=range objs{
   r,e:=c.BlockCreate(ctx,&pb.RpcBlockCreateRequest{ContextId:page,TargetId:prev,Position:model.Block_Bottom,
    Block:&model.Block{Content:&model.BlockContentOfLink{Link:&model.BlockContentLink{
     TargetBlockId:o,Style:model.BlockContentLink_Page,CardStyle:model.BlockContentLink_Card}}}})
   if e!=nil{return e}; prev=r.BlockId; fmt.Printf("card %s\n",o)
  }
  return nil
 })
 if err!=nil{fmt.Fprintln(os.Stderr,"ERROR:",err);os.Exit(1)}
}
