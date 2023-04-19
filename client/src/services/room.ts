import {
  IRoom,
  IRoomChannel,
  IRoomMessage,
} from "../interfaces/GeneralInterfaces";
import { makeRequest } from "./makeRequest";

export const createRoom = (name: string, isPrivate: boolean) =>
  makeRequest("/api/room", {
    method: "POST",
    data: { name, private: isPrivate },
  });

export const updateRoom = (id: string, name: string, isPrivate: boolean) =>
  makeRequest(`/api/room/${id}`, {
    method: "PATCH",
    data: { name, private: isPrivate },
  });

export const updateRoomChannel = ({
  name,
  main,
  id,
}: {
  name: string;
  main: boolean;
  id: string;
}) =>
  makeRequest(`/api/room/channel/${id}`, {
    method: "PATCH",
    data: { name, main },
  });

export const createRoomChannel = ({
  name,
  main,
  roomId,
}: {
  name: string;
  main: boolean;
  roomId: string;
}) =>
  makeRequest(`/api/room/${roomId}/channels`, {
    method: "POST",
    data: { name, main },
  });

export const deleteRoomChannel = (id: string) =>
  makeRequest(`/api/room/channel/${id}`, {
    method: "DELETE",
  });

export const searchRooms = (
  name: string,
  page: number
): Promise<{ rooms: IRoom[] | null; count: number }> =>
  makeRequest(`/api/rooms/search?page=${page}`, {
    data: { name },
    method: "POST",
  });

// getRooms gets all the rooms the user owns or is a member of (without pagination, since the result is small)
export const getRooms = (): Promise<IRoom[] | null> =>
  makeRequest("/api/rooms");

// getRoomsPage gets all the rooms that are public, the user owns or is member of, and rooms the user isn't banned from
export const getRoomsPage = (
  page: number
): Promise<{ rooms: IRoom[] | null; count: number }> =>
  makeRequest(`/api/rooms/all?page=${page}`, { method: "POST" });

export const getRoom = (id: string): Promise<IRoom> =>
  makeRequest(`/api/room/${id}`);

export const deleteRoom = (id: string) =>
  makeRequest(`/api/room/${id}`, { method: "DELETE" });

export const getRoomImage = (id: string) =>
  makeRequest(`/api/room/${id}/img`, { responseType: "arraybuffer" });

export const getRoomChannels = (id: string): Promise<IRoomChannel[] | null> =>
  makeRequest(`/api/room/channels/${id}`);

export const getRoomChannel = (
  id: string
): Promise<{
  messages: IRoomMessage[] | null;
  users_in_webrtc: string[] | null;
}> => makeRequest(`/api/room/channel/${id}`);

export const uploadRoomImage = (id: string, file: File) => {
  const data = new FormData();
  data.append("file", file);
  return makeRequest(`/api/room/${id}/img`, {
    method: "POST",
    data,
  });
};
