import { getAssetPath } from "@/core/helpers/assets";

const contacts = [
  {
    name: "Melody Macy",
    email: "melody@altbox.com",
    time: "1 week",
    color: "danger",
  },
  {
    image: getAssetPath("/media/avatars/300-1.jpg"),
    name: "Max Smith",
    email: "max@kt.com",
    time: "5 hrs",
    color: "danger",
  },
  {
    image: getAssetPath("/media/avatars/300-5.jpg"),
    name: "Sean Bean",
    email: "sean@dellito.com",
    time: "20 hrs",
    color: "danger",
  },
  {
    image: getAssetPath("/media/avatars/300-25.jpg"),
    name: "Brian Cox",
    email: "brian@exchange.com",
    time: "2 weeks",
    online: true,
    color: "danger",
  },
  {
    name: "Mikaela Collins",
    email: "mikaela@pexcom.com",
    time: "5 hrs",
    online: true,
    color: "warning",
  },
  {
    image: getAssetPath("/media/avatars/300-9.jpg"),
    name: "Francis Mitcham",
    email: "f.mitcham@kpmg.com.au",
    time: "20 hrs",
    online: true,
    color: "danger",
  },
  {
    name: "Olivia Wild",
    email: "olivia@corpmail.com",
    time: "3 hrs",
    color: "danger",
  },
  {
    name: "Neil Owen",
    email: "owen.neil@gmail.com",
    time: "3 hrs",
    color: "primary",
  },
  {
    image: getAssetPath("/media/avatars/300-23.jpg"),
    name: "Dan Wilson",
    email: "dam@consilting.com",
    time: "3 hrs",
    color: "danger",
  },
  {
    name: "Emma Bold",
    email: "emma@intenso.com",
    time: "1 week",
    online: true,
    color: "danger",
  },
];

export default contacts;
