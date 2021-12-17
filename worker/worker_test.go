package worker

// func Test_Queue_Mouse(t *testing.T) {
// 	keyWord := "mouse"
// 	cleanupCtx := context.Background()
// 	pProduct := make(chan pb.UserResponse, 1000)
// 	results := []pb.UserResponse{}

// 	go func() {
// 		for product := range pProduct {
// 			results = append(results, product)
// 		}
// 	}()

// 	Queue(cleanupCtx, keyWord, pProduct)

// 	if len(results) == 0 {
// 		t.Error("error in Queue, result: ", results)
// 	}

// }
