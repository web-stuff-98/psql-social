import { defineStore } from "pinia";

type SocketStoreState = {
  socket?: WebSocket;
};

const useSocketStore = defineStore("socket", {
  state: () =>
    ({
      socket: undefined,
    } as SocketStoreState),
  actions: {
    send(data: string | object) {
      if (this.socket?.readyState === 1)
        this.socket.send(
          typeof data === "object" ? JSON.stringify(data) : data
        );
      else console.warn("Socket unavailable");
    },
    async connectSocket(token: string) {
      return new Promise<WebSocket>((resolve, reject) => {
        const socket = new WebSocket(
          process.env.NODE_ENV === "development" ||
          window.location.origin === "https://localhost:8080/"
            ? `ws://localhost:8080/api/ws?token=${token}`
            : `wss://psql-social.herokuapp.com/api/ws?token=${token}`
        );
        socket.onopen = () => {
          resolve(socket);
        };
        socket.onerror = (e) => {
          reject(e);
        };
        socket.onclose = () => {
          this.socket = undefined;
        };
      });
    },
  },
});

export default useSocketStore;
