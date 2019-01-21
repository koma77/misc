// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/s3/s3_copy_object.go

package main

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/awserr"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "fmt"
    "sync"
    "os"
    "context"
    "sync/atomic"
    "time"
)

type Copier struct {
  cl_src  *s3.S3
  cl_dst  *s3.S3
  src     string
  dst     string
  prefix  string
  ok      uint64
  err     uint64
}

func (c *Copier) cp(key string) {
  _, err := c.cl_dst.CopyObject(&s3.CopyObjectInput{
      Bucket: aws.String(c.dst),
      CopySource: aws.String(c.src + "/" + key),
      Key: aws.String(key),
  })
  if err != nil {
    fmt.Fprintf(os.Stderr, "Unable to copy item from bucket %q to bucket %q, %v\n", c.src, c.dst, err)
    atomic.AddUint64(&c.err, 1)
    return
  }

  // Wait to see if the item got copied
  err = c.cl_dst.WaitUntilObjectExists(&s3.HeadObjectInput{Bucket: aws.String(c.dst), Key: aws.String(key)})
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error occurred while waiting for item %q to be copied to bucket %q, %v\n", key, c.dst, err)
    atomic.AddUint64(&c.err, 1)
    return
  }

  //fmt.Fprintf(os.Stderr, "Item %q successfully copied from bucket %q to bucket %q\n", key, c.src, c.dst)
  atomic.AddUint64(&c.ok, 1)
  //fmt.Printf(".")
}


func (c *Copier) bucketCopy(concurrency int) {
  keysChan := make(chan string, concurrency)
  wg := new(sync.WaitGroup)

  // Spawn workers which receives a key from keysChan and copy the object.
  for i := 0; i < concurrency; i++ {
    wg.Add(1)
    go func() {
       // Decrement the counter when the goroutine completes.
      defer wg.Done()
      // for range terminates when the keysChan is closed by the ListObjectsV2Pages.
      for key := range keysChan {
        c.cp(key)
      }
    }()
  }

  req := &s3.ListObjectsInput{
    Bucket: aws.String(c.src),
    Delimiter: aws.String("/"),
    Prefix: aws.String(c.prefix),
  }

  err := c.cl_src.ListObjectsPages(req, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
        //fmt.Println("Page,", i)
        //i++
        for _, obj := range p.Contents {
          keysChan <- *obj.Key
        }
        return true
  })

  if err != nil {
    fmt.Fprintf(os.Stderr, "unable to find bucket or %s's region not found\n", c.src)
    return
  }

  close(keysChan)
  wg.Wait()
}


func getBucketRegion(bucket string) string {
  sess := session.Must(session.NewSession())
  ctx :=  context.Background()
  region, err := s3manager.GetBucketRegion(ctx, sess, bucket, "ap-south-1")
  if err != nil {
     if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
        fmt.Fprintf(os.Stderr, "unable to find bucket %s's region not found\n", bucket)
     }
  }
  return region
}

func getBucketClient(bucket string) *s3.S3  {
  sess := session.Must(session.NewSession())
  ctx :=  context.Background()
  region, err := s3manager.GetBucketRegion(ctx, sess, bucket, "ap-southeast-1")
  if err != nil {
     if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
        fmt.Fprintf(os.Stderr, "unable to find bucket %s's region not found\n", bucket)
     }
     return nil
  }

  s := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
  return  s3.New(s)
}

/* 
func bucketList(bucket string) {
  r := getBucketRegion(bucket)
  sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(r)}))
  svc := s3.New(sess)

  //i := 0
  err := svc.ListObjectsPages(&s3.ListObjectsInput{
      Bucket: &bucket,
      Delimiter: aws.String("/"),
      Prefix: aws.String("live/customer-ids/"),
    }, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
        //fmt.Println("Page,", i)
        //i++
        for _, obj := range p.Contents {
          fmt.Println("Object:", *obj.Key)
        }
        return true
  })
  if err != nil {
    fmt.Fprintf(os.Stderr, "unable to find bucket or %s's region not found\n", bucket)
    return
  }
}
*/

func NewCopier(src, prefix, dst string) *Copier {
  c1 := &Copier{
    cl_src : getBucketClient(src),
    cl_dst : getBucketClient(dst),
    src: src,
    dst: dst,
    prefix: prefix,
  }

  return c1
}


func main() {
  if len(os.Args) != 4 {
      fmt.Fprintf(os.Stderr, "Source bucket, prefix, and destination bucket names required\nUsage: %s src_bucket prefix dst_bucket\nPrefix can be \"\"", os.Args[0])
      os.Exit(1)
  }

  src := os.Args[1]
  prefix := os.Args[2]
  dst := os.Args[3]


  cc1 := NewCopier(src, prefix, dst)

  ticker := time.NewTicker(2000 * time.Millisecond)
  go func() {
    for _ = range ticker.C {
      ok := atomic.LoadUint64(&cc1.ok)
      err := atomic.LoadUint64(&cc1.err)
      fmt.Printf("\rOK: %d ERR: %d", ok, err)
    }
  }()

  cc1.bucketCopy(300)
}
