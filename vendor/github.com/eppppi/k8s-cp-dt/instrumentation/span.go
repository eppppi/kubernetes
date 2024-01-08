package instrumentation

import (
	"context"
	"fmt"
	"log"
	"time"

	mergelogpb "github.com/eppppi/k8s-cp-dt/mergelog/src/pkg/grpc"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type contextKey string

const (
	KOC_PARENTID_KEY contextKey = "eppppi.github.io/koc-parentid"
)

func GetParentId(ctx context.Context) string {
	if val := ctx.Value(KOC_PARENTID_KEY); val == nil {
		return ""
	} else {
		return val.(string)
	}
}

func SetParentId(ctx context.Context, parentId string) context.Context {
	return context.WithValue(ctx, KOC_PARENTID_KEY, parentId)
}

var (
	spanCh     chan *mergelogpb.Span
	mergelogCh chan *mergelogpb.Mergelog
)

const (
	CHANNEL_SIZE = 100
)

// InitSender initializes a sender (gRPC client).
// If wait is true, this func waits until setup is done.
func InitSender(endpoint string) (<-chan error, func()) {
	doneCh := make(chan struct{})
	spanCh = make(chan *mergelogpb.Span, CHANNEL_SIZE)
	mergelogCh = make(chan *mergelogpb.Mergelog, CHANNEL_SIZE)
	setupDoneCh := make(chan error)
	finishCh := make(chan struct{})
	go runSender(doneCh, endpoint, spanCh, mergelogCh, setupDoneCh, finishCh)

	return setupDoneCh, func() {
		doneCh <- struct{}{}
		// wait until sender is shutdown
		<-finishCh
	}
}

// Span is a span that is to be converted to the Span struct of protobuf
type Span struct {
	cpid       string
	startTime  time.Time
	endTime    time.Time
	service    string
	objectKind string
	objectName string
	message    string
	spanId     string
	parentId   string
}

// Start starts a span. If returned span is nil, no span is started.
func Start(ctx context.Context, cpid, service, objKind, objName, msg string) (context.Context, *Span) {
	// validate cpid is not empty string
	if cpid == "" {
		return ctx, nil
	}

	// 古いctxには、呼び出し側の関数のspanIdが入っている
	// newCtxには、出力されるspanと同じ情報が入っている
	spanId, _ := uuid.NewRandom()
	span := &Span{
		cpid:       cpid,
		startTime:  time.Now(),
		service:    service,
		objectKind: objKind,
		objectName: objName,
		message:    msg,
		spanId:     spanId.String(),  // 新しいspanIdを入れる
		parentId:   GetParentId(ctx), // 古いctxのspanIdを入れる
	}
	newCtx := SetParentId(ctx, spanId.String())
	return newCtx, span
}

// End ends a span
func (s *Span) End() {
	s.endTime = time.Now()
	// push to channel
	spanCh <- s.ToProtoSpan()
	fmt.Println("span end")
}

// GenerateNewTctxAndSendMergelog generates a new trace context. if retTctx is nil, no mergelog is sent.
func MergeAndSendMergelog(newTctx *TraceContext, sourceTctxs []*TraceContext, causeMsg, by string) (*TraceContext, error) {
	return mergeAndSendMergelog(newTctx, sourceTctxs, causeMsg, by)
}

// generateNewTctxAndSendMergelog generates a new trace context. if retTctx is nil, no mergelog is sent.
func mergeAndSendMergelog(newTctx *TraceContext, sourceTctxs []*TraceContext, causeMsg, by string) (*TraceContext, error) {
	// validate arguments
	if err := newTctx.validateTctx(); err != nil {
		return nil, err
	}
	// deep-copy tctxs so that the original tctxs are not modified
	newTctx = newTctx.DeepCopyTraceContext()
	newSourceTctxs := make([]*TraceContext, 0)
	for i := 0; i < len(sourceTctxs); i++ {
		if err := sourceTctxs[i].validateTctx(); err != nil {
			log.Println("validation error, skipping this tctx:", err)
		} else {
			newSourceTctxs = append(newSourceTctxs, sourceTctxs[i].DeepCopyTraceContext())
		}
	}
	if len(newSourceTctxs) == 0 {
		log.Println("size of valid sourceTctxs is 0, so no need to merge and no mergelog is sent")
		return newTctx, nil
	}

	retTctx, newCpid, sourceCpids := mergeTctxs(append(newSourceTctxs, newTctx))
	if sourceCpids != nil {
		err := sendMergelog(newCpid, sourceCpids, mergelogpb.CauseType_CAUSE_TYPE_MERGE, causeMsg, by)
		if err != nil {
			panic(err) // should not happen because of prior validation
		}
	}
	return retTctx, nil
}

// GenerateAndSendMergelog generates a mergelog and push it to channel
func sendMergelog(newCpid string, sourceCpids []string, causeType mergelogpb.CauseType, causeMsg, by string) error {
	// validate cpids
	if newCpid == "" {
		return fmt.Errorf("newCpid is empty string")
	}
	for _, sourceCpid := range sourceCpids {
		if sourceCpid == "" {
			return fmt.Errorf("one of sourceCpid is empty string")
		}
	}

	srcCpids := make([]*mergelogpb.CPID, 0)
	for _, cpid := range sourceCpids {
		srcCpids = append(srcCpids, &mergelogpb.CPID{Cpid: cpid})
	}
	mergelog := &mergelogpb.Mergelog{
		NewCpid:      &mergelogpb.CPID{Cpid: newCpid},
		SourceCpids:  srcCpids,
		Time:         timestamppb.New(time.Now()),
		CauseType:    mergelogpb.CauseType_CAUSE_TYPE_NEW_CHANGE,
		CauseMessage: causeMsg,
		By:           by,
	}
	mergelogCh <- mergelog
	return nil
}

// ToProtoSpan converts a span to the Span struct of protobuf
func (s *Span) ToProtoSpan() *mergelogpb.Span {
	return &mergelogpb.Span{
		Cpid:       &mergelogpb.CPID{Cpid: s.cpid},
		StartTime:  timestamppb.New(s.startTime),
		EndTime:    timestamppb.New(s.endTime),
		Service:    s.service,
		ObjectKind: s.objectKind,
		ObjectName: s.objectName,
		Message:    s.message,
		SpanId:     s.spanId,
		ParentId:   s.parentId,
	}
}

// RunSender runs a sender.
// This func is intended to be called as a goroutine.
// ctx is a context that is used to stop this func.
func runSender(doneCh <-chan struct{}, endpoint string, spanCh <-chan *mergelogpb.Span, mergelogCh <-chan *mergelogpb.Mergelog, setupDoneCh chan<- error, finishCh chan<- struct{}) {
	log.Println("runSender() started")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// TODO(improve): 毎回送信するのではなく、一定時間ごとに送信するようにする
	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		setupDoneCh <- fmt.Errorf("connection failed: %v: the trace-server is not running or the endpoint is wrong", err)
		finishCh <- struct{}{}
		return
	} else {
		log.Println("Connection succeeded")
	}
	defer conn.Close()
	client := mergelogpb.NewMergelogServiceClient(conn)

	setupDoneCh <- nil

	for {
		select {
		case <-doneCh:
			// TODO: graceful shutdown (wait until all channels are empty)
			log.Println("finishing sender")
			finishCh <- struct{}{}
			return
		case span := <-spanCh:
			req := &mergelogpb.PostSpansRequest{
				Spans: []*mergelogpb.Span{span},
			}
			_, err := client.PostSpans(context.Background(), req)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("span sent")
			}
		case mergelog := <-mergelogCh:
			req := &mergelogpb.MergelogRequest{
				Mergelogs: []*mergelogpb.Mergelog{mergelog},
			}
			_, err := client.PostMergelogs(context.Background(), req)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("mergelog sent")
			}
		}
	}
}
