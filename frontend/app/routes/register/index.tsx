import {Outlet} from "react-router";
import {
  Breadcrumb,
  BreadcrumbItem, BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator
} from "~/components/ui/breadcrumb";
import {type ReactNode, useState} from "react";

export default function Register() {
  //TODO: Сделать чтобы он ставился от аддреса
  const [step, setStep] = useState(4)

  //TODO: После подтверждения если email уже был в базе то или входит сразу или кидает на вход с ошибкой типо email привязан и кнопка восстановить акк/поменять пароль или просто зайти в auth наверное
  const stepMap: Record<number, string> = {
    1: "Регистрация",
    2: "Подтверждение",
    3: "Ввод пароля",
    4: "Галочки лицензии"
  }
  //TODO: Если не доделан этап 4 то при заходе в регестрации сразу кидает к нему и пока не подтвердит незя ничего, акк не валиден, сделан наверное в go проверку на валидность акка (по проверки документации) и ошибку почему
  const breadcrumb:ReactNode[] = [];

  for (let i = 1; i < step; i++) {
    breadcrumb.push(<BreadcrumbItem><BreadcrumbLink>{stepMap[i]}</BreadcrumbLink></BreadcrumbItem>)
    breadcrumb.push(<BreadcrumbSeparator />)
  }
  breadcrumb.push(<BreadcrumbItem><BreadcrumbPage>{stepMap[step]}</BreadcrumbPage></BreadcrumbItem>)

  const breadcrumbDivClass = "hidden lg:block lg:flex flex-row w-screen justify-center pb-3"
  return (
      <div className="flex lg:flex-col lg:justify-center lg:h-screen w-screen">
        <div className={step !== 1 ? "" + breadcrumbDivClass : "invisible" + breadcrumbDivClass}>
          <Breadcrumb>
            <BreadcrumbList>{breadcrumb}</BreadcrumbList>
          </Breadcrumb>
        </div>
        <div className="flex flex-row w-screen justify-center">
            <Outlet />
        </div>
      </div>
)
}
