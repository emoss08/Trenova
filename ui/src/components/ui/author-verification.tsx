/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { AUTHOR_NAME } from "@/constants/env";
import { Card } from "./card";
import { LazyImage } from "./image";

export function AuthorVerification() {
  if (AUTHOR_NAME !== "Eric Moss") {
    return (
      <div className="flex flex-col items-center justify-center h-screen bg-[url('https://cdn.ebaumsworld.com/2019/09/12/012018/86066195/clown-pics-and-memes25.jpg')]">
        <LazyImage
          src="https://media4.giphy.com/media/v1.Y2lkPTc5MGI3NjExZDhwam5hYjB3bHJmZnQzaWY2NGtmNmk0ajk5d2M3d2VmZGg2bjMwcyZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/xUOrvXaejeo9gMiArS/giphy.gif"
          alt="Really bro?"
          width={500}
          height={500}
        />
        <Card>
          <h1 className="text-4xl font-bold text-center text-blue-500">
            Hey friend, uhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh...........
            <br />I see you tried to claim this code as your own... LOL.
            <br />
            No worries, I&apos;ll help you revert it back. Just don&apos;t do it
            again.
            <br />
            If you use your jelly fishing glasses you&apos;ll see a command
            below this.
            <br />
            <pre className="text-[5px]">git reset HEAD^</pre>
          </h1>
        </Card>
      </div>
    );
  }

  return null;
}
