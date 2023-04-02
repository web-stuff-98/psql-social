import {
  IRoom,
  IRoomChannel,
  IRoomMessage,
} from "../interfaces/GeneralInterfaces";
import { makeRequest } from "./makeRequest";

export const createRoom = ({
  name,
  isPrivate,
}: {
  name: string;
  isPrivate: boolean;
}) =>
  makeRequest("/api/room", {
    method: "POST",
    data: { name, private: isPrivate },
  });

export const updateRoom = ({
  name,
  isPrivate,
  id,
}: {
  name: string;
  isPrivate: boolean;
  id: string;
}) =>
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

export const getRooms = (): Promise<IRoom[] | null> =>
  makeRequest("/api/rooms");

export const getRoom = (id: string): Promise<IRoom> =>
  makeRequest(`/api/room/${id}`);

export const deleteRoom = (id: string) =>
  makeRequest(`/api/room/${id}`, { method: "DELETE" });

export const getRoomChannels = (id: string): Promise<IRoomChannel[] | null> =>
  makeRequest(`/api/room/channels/${id}`);

export const getRoomChannel = (id: string): Promise<IRoomMessage[] | null> =>
  makeRequest(`/api/room/channel/${id}`);
