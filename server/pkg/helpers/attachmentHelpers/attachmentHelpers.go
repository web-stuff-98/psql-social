package attachmenthelpers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetTableNames(conn *pgxpool.Conn, ctx context.Context, id string) (metaTable string, chunkTable string, err error) {
	var isDirectMessage, isRoomMsg bool
	if selectDirectMessage, err := conn.Conn().Prepare(ctx, "attachment_server_chunk_loop_select_direct_messages_stmt", "SELECT EXISTS(SELECT 1 FROM direct_messages WHERE id = $1);"); err != nil {
		return "", "", err
	} else {
		if err = conn.Conn().QueryRow(ctx, selectDirectMessage.Name, id).Scan(&isDirectMessage); err != nil {
			return "", "", err
		}
	}
	if selectRoomMessage, err := conn.Conn().Prepare(ctx, "attachment_server_chunk_loop_select_room_messages_stmt", "SELECT EXISTS(SELECT 1 FROM room_messages WHERE id = $1);"); err != nil {
		return "", "", err
	} else {
		if err = conn.Conn().QueryRow(ctx, selectRoomMessage.Name, id).Scan(&isRoomMsg); err != nil {
			return "", "", err
		}
	}
	if isDirectMessage {
		return "direct_message_attachment_metadata", "direct_message_attachment_chunks", nil
	}
	if isRoomMsg {
		return "room_message_attachment_metadata", "room_message_attachment_chunks", nil
	}
	return "", "", fmt.Errorf("Message not found in either table")
}
