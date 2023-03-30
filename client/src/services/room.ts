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
