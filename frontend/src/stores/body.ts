import { ref } from "vue";
import { defineStore } from "pinia";

export const useBodyStore = defineStore("body", () => {
  const classes = ref<any>({});

  function getClasses(key: string) {
    return classes.value[key];
  }

  function addBodyClassname(className: string) {
    document.body.classList.add(className);
  }

  function removeBodyClassName(className: string) {
    document.body.classList.remove(className);
  }

  function addBodyAttribute(payload: { qualifiedName: string; value: string }) {
    const { qualifiedName, value } = payload;
    document.body.setAttribute(qualifiedName, value);
  }

  function removeBodyAttribute(payload: { qualifiedName: string }) {
    const { qualifiedName } = payload;
    document.body.removeAttribute(qualifiedName);
  }

  function addClassname(payload: { position: string; className: string }) {
    const { position, className } = payload;
    if (!classes.value[position]) {
      classes.value[position] = [className];
    } else {
      classes.value[position].push(className);
    }
  }

  return {
    getClasses,
    addBodyClassname,
    removeBodyClassName,
    addBodyAttribute,
    removeBodyAttribute,
    addClassname,
  };
});
