package main
import ("context";"fmt";"os";"github.com/anyproto/anytype-heart/pb";"github.com/anyproto/anytype-heart/pb/service";"github.com/anyproto/anytype-cli/core")
func main(){
 _=core.GRPCCall(func(ctx context.Context,c service.ClientCommandsClient)error{
  s,e:=c.ObjectShow(ctx,&pb.RpcObjectShowRequest{ObjectId:os.Args[1]});if e!=nil{return e}
  for _,det:=range s.ObjectView.GetDetails(){
   for k:=range det.GetDetails().GetFields(){
    switch k {case "coverId","coverType","iconImage","iconEmoji": fmt.Printf("DETAIL %s\n",k)}
   }
  }
  for _,b:=range s.ObjectView.GetBlocks(){
   switch {
   case b.GetText()!=nil: fmt.Printf("text/%v\n",b.GetText().Style)
   case b.GetLink()!=nil: fmt.Println("LINK-to-object block")
   case b.GetDataview()!=nil: fmt.Println("DATAVIEW (inline set/board)")
   case b.GetLayout()!=nil: fmt.Printf("LAYOUT/%v\n",b.GetLayout().Style)
   case b.GetFile()!=nil: fmt.Println("file/media")
   case b.GetDiv()!=nil: fmt.Println("divider")
   case b.GetBookmark()!=nil: fmt.Println("bookmark")
   case b.GetRelation()!=nil: fmt.Println("relation-block")
   case b.GetTableOfContents()!=nil: fmt.Println("toc")
   }
  }
  return nil
 })
}
