// appendmd <pageId> <mdFile> — append markdown (parsed into blocks) to a page
// via BlockPaste's text slot. Habit tool for logging lessons. gRPC.
package main
import ("context";"fmt";"os"
 "github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service";"github.com/anyproto/anytype-cli/core")
func main(){
 page:=os.Args[1]; md,e0:=os.ReadFile(os.Args[2]); if e0!=nil{fmt.Fprintln(os.Stderr,e0);os.Exit(1)}
 err:=core.GRPCCall(func(ctx context.Context,c service.ClientCommandsClient)error{
  _,e:=c.BlockPaste(ctx,&pb.RpcBlockPasteRequest{ContextId:page,TextSlot:string(md)})
  if e!=nil{return e}; fmt.Println("appended"); return nil
 })
 if err!=nil{fmt.Fprintln(os.Stderr,"ERROR:",err);os.Exit(1)}
}
