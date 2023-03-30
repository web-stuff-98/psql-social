import { IRoom } from "../interfaces/GeneralInterfaces";
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

export const getRooms = (): Promise<IRoom[] | null> =>
  makeRequest("/api/rooms");

export const getRoom = (id: string): Promise<IRoom> =>
  makeRequest(`/api/room/${id}`);

export const deleteRoom = (id: string) =>
  makeRequest(`/api/room/${id}`, { method: "DELETE" });
