import { IUser } from "../interfaces/GeneralInterfaces";
import { makeRequest } from "./makeRequest";

export const getUserBio = (id: string): Promise<string> =>
  makeRequest(`/api/user/bio/${id}`);

export const getUser = (id: string): Promise<IUser> =>
  makeRequest(`/api/user/${id}`);

export const getUserPfp = (id: string) =>
  makeRequest(`/api/user/pfp/${id}`, { responseType: "arraybuffer" });

export const getUserByName = (username: string): Promise<string> =>
  makeRequest(`/api/user/name`, { data: { username }, method: "POST" });
