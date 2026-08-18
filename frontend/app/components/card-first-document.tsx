import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "@/components/ui/card.tsx"
import { Button } from "@/components/ui/button.tsx"
import { useNavigate } from "react-router-dom"

export function CardFirstDocument() {
  const navigate = useNavigate()
  return (
    <Card className="m-4 h-full w-full lg:w-lg">
      <CardHeader>
        <CardTitle>Обзательно к прочтению</CardTitle>
        <CardDescription>
          Сдесь описано правильно без соглашения которых дальше нельзя
          использовать сайт
        </CardDescription>
      </CardHeader>
      <CardContent className="my-3">
        Lorem ipsum dolor sit amet, consectetur adipisicing elit. Ad aliquam
        animi asperiores assumenda atque blanditiis consequuntur culpa delectus
        dolores dolorum eius eos explicabo facilis fuga id laboriosam, laborum
        magni nam nihil omnis possimus provident quis quisquam saepe sit tempore
        temporibus tenetur, totam ullam voluptate! Aperiam blanditiis distinctio
        dolorem facere in quod? Ab accusamus beatae blanditiis cum cumque
        debitis dicta esse eum, excepturi id impedit iusto laboriosam, magni
        modi, nostrum odit officiis porro quae quasi qui quia quod ratione
        recusandae sequi sit totam velit voluptate voluptatem voluptates
        voluptatum? Aliquam aperiam consequuntur esse explicabo sunt? Fuga
        laudantium libero quibusdam? Facere harum impedit pariatur quas rem.
        Adipisci alias aperiam, asperiores aut consectetur consequuntur corporis
        est et facere hic inventore iste, nesciunt nihil numquam possimus
        praesentium quos sint ullam, vel voluptatum? Amet, consectetur doloribus
        eligendi esse facilis numquam porro quis saepe vel voluptatum. Adipisci
        dicta doloribus minima necessitatibus omnis, pariatur quaerat quod sed?
        Dignissimos esse nemo quae. Asperiores, corporis dolor error maxime
        tempore unde voluptates. A ab adipisci aliquam aperiam, architecto culpa
        cupiditate, debitis doloribus eligendi iste iusto obcaecati officia quam
        quo sequi sunt tempora. Alias, aliquid aut enim molestias neque non
        officia perspiciatis totam ullam voluptatem. Aut cum delectus deserunt
        maiores nemo nihil nisi non quae quaerat vero. Commodi culpa delectus
        error iste mollitia neque nostrum veritatis! Aut consequatur doloribus
        labore maxime ratione! A adipisci aliquam atque consequuntur corporis
        culpa, cumque, cupiditate deserunt, fuga ipsam nesciunt pariatur
        possimus provident quaerat quo reiciendis repellat reprehenderit
        similique totam veritatis. Ad alias at dolorum eveniet molestias quo
        quos reiciendis voluptates! A atque ea hic ipsa minima, molestias natus
        pariatur quam, quibusdam quis quisquam sunt tenetur velit. Assumenda
        consequuntur dignissimos distinctio et fugit nam nostrum omnis quia
        recusandae, repellat tempore vero voluptas voluptatibus! Animi corporis,
        doloribus excepturi illo itaque iure, labore molestias nobis nostrum
        quidem quod ratione sint suscipit! Accusamus alias dignissimos, facilis
        obcaecati quos unde velit. Alias corporis cupiditate deserunt dolore hic
        itaque, labore laboriosam, laborum maiores molestias nostrum quas quis
        quod repellat similique sit tempora temporibus totam velit voluptates.
        Ab asperiores atque aut beatae blanditiis consequuntur corporis culpa
        cupiditate dolore eligendi error et ex excepturi exercitationem facere
        facilis fugit hic incidunt iusto labore laudantium minima modi mollitia,
        nisi nostrum obcaecati, omnis perspiciatis porro quisquam quod rem
        repudiandae sint tempore temporibus voluptate voluptates voluptatum?
        Aperiam, commodi consequatur dolorum ea est facere natus nisi nostrum
        odit officia quaerat repellat veniam, vitae! Laborum rerum, voluptatum.
        Architecto dolor error fugit ipsum minus molestias odio perferendis
        quasi qui ratione, saepe sapiente soluta! Ab architecto cum cumque
        ducimus ipsam mollitia nemo numquam officia, perspiciatis qui ratione
        rerum sequi sit? Accusamus cumque ipsa nobis quo tempora. Ad alias
        asperiores consequuntur corporis, deserunt dicta earum exercitationem
        facilis iste itaque, laborum maxime nihil quis quo sint soluta totam!
        Aliquam deserunt doloribus est facere id ipsa magnam nemo qui, rem sed.
        Accusamus assumenda deserunt fuga nemo. Adipisci aliquam aliquid
        aperiam, atque consequuntur, corporis, deserunt dolor dolores enim
        expedita facilis ipsa molestiae numquam placeat provident quae quaerat
        quasi quod quos similique ullam unde velit veritatis vitae.
      </CardContent>
      <CardFooter className="flex flex-col">
        <div className="mb-3 text-ring">
          Нажимая Пропустить или Авторизоватся вы соглашаетесь с условиями
          использования
        </div>
        <div className="flex w-full flex-row justify-around">
          {/*TODO: Сделать проброс действия чтобы или скрывал popup или перекидывал на главную страницу*/}
          <Button variant="outline" className="p-4">
            Пропустить
          </Button>
          {/*TODO: Сделать ссылку на авторизацию, с запоминанием страницы откуда*/}
          <Button className="p-4" onClick={() => navigate("/register")}>
            Авторизоватся
          </Button>
        </div>
      </CardFooter>
    </Card>
  )
}
