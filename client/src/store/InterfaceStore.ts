import { defineStore } from "pinia";

type DarkModeState = {
  darkMode: boolean;
};

const useInterface = defineStore("interface", {
  state: () =>
    ({
      darkMode: true,
    } as DarkModeState),
});

export default useInterface;
