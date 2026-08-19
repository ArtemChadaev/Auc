import {NavLink, Outlet} from "react-router";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarProvider,
  SidebarTrigger
} from "~/components/ui/sidebar";
import {Button} from "~/components/ui/button";

export default function TitlePage() {
  const login = true;
  const coins = 3;
  return (
    <SidebarProvider defaultOpen={false}>
      <div className="w-full h-sceen">
        <header className="border-b-2 w-full h-10 flex">
          <div className="w-full h-full" />
          {
            login ? <>
                <Button variant="link" className="m-auto pr-5" size='icon-lg'>{coins}&nbsp;coins</Button>
                <SidebarTrigger size="lg" className="m-auto" />
              </> : <Button variant="link" className="m-auto"><NavLink to="/register">Зайти</NavLink></Button>
          }
        </header>

        <div className="h-full w-full flex justify-center">
          <Outlet />
        </div>
      </div>
      <Sidebar side='right'>
        <SidebarHeader>
        </SidebarHeader>
        <SidebarContent>

        </SidebarContent>
        <SidebarFooter>
          <SidebarTrigger size="lg" className="my-auto mr-auto" />
        </SidebarFooter>
      </Sidebar>

    </SidebarProvider>
  )
}