package main

import (
	"context"
	"errors"
	"io"
	"log"

	"google.golang.org/grpc"
)

func myStreamClientInteceptor1(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	log.Println("[pre] my stream client interceptor 1", method)
	stream, err := streamer(ctx, desc, cc, method, opts...)
	return &myClientStreamWrapper1{stream}, err
}

type myClientStreamWrapper1 struct {
	grpc.ClientStream
}

func (s *myClientStreamWrapper1) SendMsg(m interface{}) error {
	log.Println("[pre message] my stream client interceptor 1: ", m)
	return s.ClientStream.SendMsg(m)
}

func (s *myClientStreamWrapper1) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if !errors.Is(err, io.EOF) {
		log.Println("[post message] my stream client interceptor 1: ", m)
	}
	return err
}

func (s *myClientStreamWrapper1) CloseSend() error {
	err := s.ClientStream.CloseSend()
	log.Println("[post] my stream client interceptor 1")
	return err
}

func myStreamClientInteceptor2(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	log.Println("[pre] my stream client interceptor 2", method)
	stream, err := streamer(ctx, desc, cc, method, opts...)
	return &myClientStreamWrapper2{stream}, err
}

type myClientStreamWrapper2 struct {
	grpc.ClientStream
}

func (s *myClientStreamWrapper2) SendMsg(m interface{}) error {
	log.Println("[pre message] my stream client interceptor 2: ", m)
	return s.ClientStream.SendMsg(m)
}

func (s *myClientStreamWrapper2) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if !errors.Is(err, io.EOF) {
		log.Println("[post message] my stream client interceptor 2: ", m)
	}
	return err
}

func (s *myClientStreamWrapper2) CloseSend() error {
	err := s.ClientStream.CloseSend()
	log.Println("[post] my stream client interceptor 2")
	return err
}
