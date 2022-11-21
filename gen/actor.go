/*
 * @Date: 2022-11-21 00:46:14
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2022-11-21 03:29:03
 * @FilePath: /vlgo/gen/actor.go
 * @Description: Do not edit
 */
package gen

import (
	"reflect"
	"time"
	"vlgo/ecode"
	"vlgo/utils"

	"go.uber.org/atomic"
)

const (
	stopReasonPanic = "panic"
	stopReasonDone  = "done"
	stopReasonRet   = "ret"
)

const (
	logActor = "Actor"
	logReply = "Reply"
	logSend  = "Send"
	logCast  = "Cast"
	logStart = "Start"
)

const (
	genTimeOut     = 7 * time.Second
	stopTimeOut    = 30 * time.Second
	MaxMailBoxLen  = 2000
	GenInitTimeout = time.Second * 10
)

// callNoReply 返回这个的时候，表示后续会处理然后主动call调用者，不堵塞gen_server。但是堵塞调用者
type callNoReply struct{}

// CallNoRep for no_reply
var CallNoRep *callNoReply = nil

type Actor struct {
	H ActorHandlerI

	Ctx ActorCtx
	Wt  Waiter

	Name         string
	State        interface{}
	Mailbox      chan interface{}
	InterruptBox chan time.Duration
	DefaultOut   time.Duration

	IsStopped *atomic.Bool
}

type ActorHandlerI interface {
	Init(ctx ActorCtx, msg, state interface{}) ActorRet

	Handle(ctx ActorCtx, msg interface{}, state interface{}) ActorRet
	Stop(ctx ActorCtx, msg interface{}, state interface{})
	Tick(ctx ActorCtx, state interface{}) ActorRet
	Timeout(ctx ActorCtx, state interface{}) ActorRet
}

type ActorCaller struct {
	ch chan ActorRet
}

type ActorCall struct {
	caller ActorCaller
	msg    interface{}
}

// SendReply send reply to caller
func (caller ActorCaller) SendReply(ret interface{}, err ecode.VEI) {
	log.Debugf(logActor, logReply, "direct send ret %v<-%v", caller.ch, ret)
	safeSendRet(caller.ch, NewGenRet(ret, err))
}

// sendRet send reply to caller
func (caller ActorCaller) SendRet(v ActorRet) {
	safeSendRet(caller.ch, v)
}

type ActorCast struct {
	msg interface{}
}

type ActorRet struct {
	retVal    interface{}
	vErr      ecode.VEI
	time      time.Duration
	isStopped bool
}

func (r ActorRet) ret() interface{} {
	return r.retVal
}

func (r ActorRet) err() ecode.VEI {
	return r.vErr
}

func (r ActorRet) tm() time.Duration {
	return r.time
}

func (r ActorRet) IsStopped() bool {
	return r.isStopped
}

// GenTimer used for cancel
type ActorTimer struct {
	*time.Timer
}

// ActorCtx for gen_call
type ActorCtx struct {
	name   string
	caller ActorCaller
}

func Ctx(name string) ActorCtx {
	return ActorCtx{name: name}
}

func (ctx ActorCtx) Name() string {
	return ctx.name
}

func (ctx ActorCtx) Caller() ActorCaller {
	return ctx.caller
}

// Start method
func (s *Actor) Start(ctx ActorCtx, initMsg, state interface{}, handle ActorHandlerI) (interface{}, ecode.VEI) {
	s.H = handle
	s.Ctx = ctx
	s.Name = ctx.name
	s.Mailbox = make(chan interface{}, 1000)

	s.IsStopped = atomic.NewBool(false)

	s.InterruptBox = make(chan time.Duration)
	s.State = state

	wt := NewWaiter("gen_" + s.Name)
	s.Wt = wt

	initRetCh := make(chan ActorRet)
	// enter new go routine loop
	if s.DefaultOut != 0 {
		go s.loop(nil, time.NewTimer(s.DefaultOut), initMsg, initRetCh)
	} else {
		go s.loop(nil, nil, initMsg, initRetCh)
	}

	select {
	case v := <-initRetCh:
		return v.retVal, v.vErr
	case <-time.After(GenInitTimeout):
		log.Errorf(logActor, logSend, "gen: %s init Timeout", s.Name)
		return nil, ecode.ErrActorInitTimeout
	}
}

// Cast method
func (s *Actor) Cast(msg interface{}) {
	log.Debugf("Gen", "Call", "send cast msg %v<-%v", s.Mailbox, msg)
	safeSendChan(s.Mailbox, &ActorCast{msg})
}

// Call method
func (s *Actor) Call(msg interface{}) (interface{}, ecode.VEI) {
	return s.TimeCall(msg, genTimeOut)
}

// TimeCall method
func (s *Actor) TimeCall(msg interface{}, overDuration time.Duration) (interface{}, ecode.VEI) {
	from := make(chan ActorRet, 1)
	overTimer := time.NewTimer(overDuration)
	callMsg := &ActorCall{ActorCaller{ch: from}, msg}
	log.Debugf("Gen", "Call", "from %v send call msg %v<-%v", from, s.Mailbox, msg)

	select {
	case <-overTimer.C:
		return nil, ecode.ErrActorCallTimeout

	case s.Mailbox <- callMsg:
		select {
		case <-overTimer.C:
			return nil, ecode.ErrActorHandleTimeout

		case ret := <-from:
			log.Debugf("Gen", "Call", "got call ret:%v %v<-%v", ret, from, s.Mailbox)
			//if ret == nil {
			//	return nil, ecode.ErrGenMayDown
			//}
			return ret.ret(), ret.err()
		}
	}
}

// AfterCast method  for send_after
func (s *Actor) AfterCast(tm time.Duration, msg interface{}) *ActorTimer {
	ret := time.AfterFunc(tm, func() {
		safeSendChan(s.Mailbox, &ActorCast{msg})
	})
	return &ActorTimer{ret}
}

// StartTicker method
func (s *Actor) StartTicker(tm time.Duration) {
	go safeSendTicker(s.IsStopped, s.InterruptBox, tm)
}

// StopTicker method
func (s *Actor) StopTicker() {
	go safeSendTicker(s.IsStopped, s.InterruptBox, 0)
}

func (s *Actor) loop(ticker *time.Ticker, out *time.Timer, initMsg interface{}, retChan chan ActorRet) {
	initRet := s.H.Init(s.Ctx, initMsg, s.State)
	retChan <- initRet
	stopped, ticker, out := s.handleRet(ticker, initRet)

	loop := !stopped
	for loop {
		loop, ticker, out = s.doLoop(ticker, out)
	}
}

func (s *Actor) doLoop(ticker *time.Ticker, out *time.Timer) (loop bool, tk *time.Ticker, ot *time.Timer) {
	loop, tk, ot = true, nil, nil

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Gen", "Panic", "stack: %s, err %v", utils.Stack(), r)
		}
	}()

	if ticker != nil && out != nil {
		return s.loopWithTickOut(ticker, out)
	}

	if ticker != nil && out == nil {
		return s.loopWithTick(ticker)
	}

	if ticker == nil && out != nil {
		return s.loopWithOut(out)
	}

	return s.simpleLoop()
}

func (s *Actor) loopWithTickOut(ticker *time.Ticker, out *time.Timer) (bool, *time.Ticker, *time.Timer) {
	select {
	case tm := <-s.InterruptBox:
		return s.handleInteruput(ticker, tm)

	case <-ticker.C:
		return s.handleRet(ticker, s.H.Tick(s.Ctx, s.State))

	case msg := <-s.Mailbox:
		return s.handleRet(ticker, s.handleMail(s.Ctx, msg))

	case <-out.C:
		return s.handleRet(ticker, s.H.Timeout(s.Ctx, s.State))
	}
}

func (s *Actor) loopWithTick(ticker *time.Ticker) (bool, *time.Ticker, *time.Timer) {
	select {
	case tm := <-s.InterruptBox:
		return s.handleInteruput(ticker, tm)

	case <-ticker.C:
		return s.handleRet(ticker, s.H.Tick(s.Ctx, s.State))

	case msg := <-s.Mailbox:
		return s.handleRet(ticker, s.handleMail(s.Ctx, msg))
	}
}

func (s *Actor) loopWithOut(out *time.Timer) (bool, *time.Ticker, *time.Timer) {
	select {
	case tm := <-s.InterruptBox:
		return s.handleInteruput(nil, tm)

	case msg := <-s.Mailbox:
		return s.handleRet(nil, s.handleMail(s.Ctx, msg))

	case <-out.C:
		return s.handleRet(nil, s.H.Timeout(s.Ctx, s.State))
	}
}

func (s *Actor) simpleLoop() (bool, *time.Ticker, *time.Timer) {
	select {
	case tm := <-s.InterruptBox:
		return s.handleInteruput(nil, tm)

	case msg := <-s.Mailbox:
		return s.handleRet(nil, s.handleMail(s.Ctx, msg))
	}
}

func (s *Actor) handleInteruput(ticker *time.Ticker, tm time.Duration) (bool, *time.Ticker, *time.Timer) {
	if ticker != nil {
		ticker.Stop()
	}
	if tm > 0 {
		return false, time.NewTicker(tm), s.outTimer(0)
	} else {
		return false, nil, s.outTimer(0)
	}
}

func (s *Actor) handleMail(ctx ActorCtx, msg interface{}) ActorRet {
	switch msg := msg.(type) {
	case *ActorCall:
		from, data := msg.caller, msg.msg
		ctx.caller = from
		log.Debugf("Gen", "Call", "%v got call %v<-%v", ctx.name, typeName(data), s.Mailbox)

		ret := s.H.Handle(ctx, data, s.State)
		if _, ok := ret.ret().(*callNoReply); !ok {
			log.Debugf("Gen", "Call", "%v send ret %v<-%v", ctx.name, from, ret.ret())
			from.SendRet(ret)
		}
		return ret

	case *ActorCast:
		data := msg.msg
		if msgName := typeName(data); msgName != "addLandCast" {
			log.Debugf("Gen", "Cast", "%v got cast msg %v<-%v", ctx.name, msgName, s.Mailbox)
		}
		ret := s.H.Handle(ctx, data, s.State)
		return ret

	default:
		log.Errorf("Gen", "Call", "%v unexpected gen msg %v<-%v", ctx.name, typeName(msg), s.Mailbox)
		return NewGenRet(nil, nil)
	}
}

func (s *Actor) stop(reason string) {
	if !s.IsStopped.CAS(false, true) {
		log.Errorf(logActor, logActor, "stop gen %v multi times", reflect.TypeOf(s.State).String())
		return
	}

	s.H.Stop(s.Ctx, reason, s.State)
	close(s.Mailbox)
	close(s.InterruptBox)

	log.Infof(logActor, logActor, "svr:%v stopped", reflect.TypeOf(s.State).String())
}

func (s *Actor) handleRet(ticker *time.Ticker, ret ActorRet) (bool, *time.Ticker, *time.Timer) {
	if ret.IsStopped() {
		if ticker != nil {
			ticker.Stop()
		}
		s.H.Stop(s.Ctx, stopReasonRet, s.State)
		return true, nil, nil
	}

	return false, ticker, s.outTimer(ret.tm())
}

func (s *Actor) outTimer(d time.Duration) *time.Timer {
	if d != 0 {
		return time.NewTimer(d)
	}

	if s.DefaultOut != 0 {
		return time.NewTimer(s.DefaultOut)
	}

	return nil
}

func safeSendRet(ch chan ActorRet, data ActorRet) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf(logActor, logSend, "send ret msg %v<-%v, err:%v, stack:%s", ch, data, r, utils.Stack())
		}
	}()
	ch <- data
}

func safeSendTicker(stopFlag *atomic.Bool, ch chan time.Duration, tm time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf(logActor, logSend, "send send ticker %v<-%v, err:%v, stack:%s", ch, tm, r, utils.Stack())
		}
	}()
	if stopFlag.Load() {
		return
	}
	ch <- tm
}

func spawnSafeSendChan(ch chan interface{}, data interface{}) {
	go safeSendChan(ch, data)
}

func safeSendChan(ch chan interface{}, data interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf(logActor, logReply, "send msg %v<-%v ", ch, data)
		}
	}()

	if ch != nil {
		ch <- data
	} else {
		log.Errorf(logActor, logCast, "send msg %v to nil channel", data)
	}
}

// NewGenRet create a gen sendRet
func NewGenRet(ret interface{}, err ecode.VEI) ActorRet {
	return ActorRet{retVal: ret, vErr: err, time: 0, isStopped: false}
}

// NewTimeRet create a gen time sendRet
func NewTimeRet(ret interface{}, err ecode.VEI, tm time.Duration) ActorRet {
	return ActorRet{retVal: ret, vErr: err, time: tm, isStopped: false}
}

// NewStopRet create a gen stop sendRet
func NewStopRet(ret interface{}, err ecode.VEI) ActorRet {
	return ActorRet{retVal: ret, vErr: err, time: 0, isStopped: true}
}

type contextCaller struct {
	ch chan ActorRet
}

// SendReply send reply to caller
func (caller contextCaller) SendReply(ret interface{}, err ecode.VEI) {
	log.Debugf(logActor, logReply, "direct send ret %v<-%v", caller.ch, ret)
	safeSendRet(caller.ch, NewGenRet(ret, err))
}

// sendRet send reply to caller
func (caller contextCaller) sendRet(v ActorRet) {
	safeSendRet(caller.ch, v)
}

func typeName(msg interface{}) string {
	rv := reflect.ValueOf(msg)
	if vk := rv.Kind(); vk == reflect.Pointer {
		rv = rv.Elem()
	}

	return reflect.TypeOf(rv.Interface()).Name()
}
