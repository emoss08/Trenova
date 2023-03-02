import React from "react";

interface LayoutProps {
  children: React.ReactNode;
}

function Layout(props: LayoutProps) {
  return (
    <div>
      {/* Add your header, navigation, or any other elements here */}
      <main>{props.children}</main>
      {/* Add your footer or any other elements here */}
    </div>
  );
}

export default Layout;