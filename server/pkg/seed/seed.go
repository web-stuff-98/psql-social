package seed

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nfnt/resize"
)

func GenerateSeed(userCount int, roomCount int, db *pgxpool.Pool) {
	users := []string{}
	rooms := []string{}

	// generate users
	for i := 0; i < userCount; i++ {
		uid := generateUser(i, db)
		users = append(users, uid)

		log.Println("Generated user")
	}

	// generate rooms
	for i := 0; i < roomCount; i++ {
		randInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(users))))
		if err != nil {
			log.Fatalf("Error generating random index in seed function:%v", err)
		}
		rid := generateRoom(i, users[randInt.Int64()], db)
		rooms = append(rooms, rid)

		log.Println("Generated room")
	}

	log.Println("Generated seed")
}

func generateRoom(index int, uid string, db *pgxpool.Pool) string {
	var id string
	if err := db.QueryRow(context.Background(), `
	INSERT INTO rooms (
		name,
		private,
		author_id,
		seeded
	) VALUES($1,$2,$3,TRUE) RETURNING id;
	`, fmt.Sprintf("TestRoom%v", index+1), false, uid).Scan(&id); err != nil {
		log.Fatalf("Error in generate room seed function:%v", err)
	}

	r := GetRandomRoomImage()
	var img image.Image
	var decodeErr error
	defer r.Close()
	img, decodeErr = jpeg.Decode(r)
	if decodeErr != nil {
		log.Fatalf("Decode error in seed generate random room image function:%v", decodeErr)
	}
	img = resize.Resize(400, 0, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		log.Fatalf("Encode error in seed generate random room image function:%v", err)
	}

	if _, err := db.Exec(context.Background(), `
	INSERT INTO room_pictures (
		room_id,
		mime,
		picture_data
	) VALUES($1,$2,$3);
	`, id, "image/jpeg", buf.Bytes()); err != nil {
		log.Fatalf("Decode error in seed add random room image SQL function:%v", err)
	}

	if _, err := db.Exec(context.Background(), `
	INSERT INTO room_channels (
		room_id,
		name,
		main,
	) VALUES($1,$2,$3);
	`, id, "Main channel", true); err != nil {
		log.Fatalf("Error in seed add room main channel SQL function:%v", err)
	}

	return id
}

func generateUser(index int, db *pgxpool.Pool) string {
	var id string
	if err := db.QueryRow(context.Background(), `
	INSERT INTO users (
		username,
		role,
		seeded,
		password
	) VALUES($1,$2,TRUE,'$2a$12$0/nq/i75R27Tu2dgqvSEiuKCrcmB20ibWIbbPEseG72Brt8yGeAkG') RETURNING id;
	`, fmt.Sprintf("TestAcc%v", index+1), "USER").Scan(&id); err != nil {
		log.Fatalf("Error in generate user seed function:%v", err)
	}

	r := GetRandomPfp()
	var img image.Image
	var decodeErr error
	defer r.Close()
	img, decodeErr = jpeg.Decode(r)
	if decodeErr != nil {
		log.Fatalf("Decode error in seed generate random pfp function:%v", decodeErr)
	}
	img = resize.Resize(180, 0, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		log.Fatalf("Encode error in seed generate random pfp function:%v", err)
	}

	if _, err := db.Exec(context.Background(), `
	INSERT INTO profile_pictures (
		user_id,
		mime,
		picture_data
	) VALUES($1,$2,$3);
	`, id, "image/jpeg", buf.Bytes()); err != nil {
		log.Fatalf("Decode error in seed add random pfp SQL function:%v", err)
	}

	return id
}

func GetRandomPfp() io.ReadCloser {
	_, err := url.Parse("https://100k-faces.glitch.me/random-image")
	if err != nil {
		log.Fatal("Failed to parse image url")
	}
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.URL.Opaque = req.URL.Path
			return nil
		},
	}
	resp, err := client.Get("https://100k-faces.glitch.me/random-image")
	if err != nil {
		log.Fatal(err)
	}
	return resp.Body
}

func GetRandomRoomImage() io.ReadCloser {
	_, err := url.Parse("https://picsum.photos/500/300")
	if err != nil {
		log.Fatal("Failed to parse image url")
	}
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.URL.Opaque = req.URL.Path
			return nil
		},
	}
	resp, err := client.Get("https://picsum.photos/500/300")
	if err != nil {
		log.Fatal(err)
	}
	return resp.Body
}
